package detect

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"

	"github.com/jcl80/dredge4us/lib/fourchan"
)

// classifications is the fixed taxonomy LLMClassifier sorts threads
// into. NONE means the thread doesn't clearly fit any of the others —
// most threads land here, and no Finding is produced for those.
var classifications = []string{
	"ARTIFACT_DROP", "CAPABILITY_CLAIM", "LEAK_DOC", "TOOLING",
	"MISUSE_DEMAND", "CORP_NEWS", "SENTIMENT_SHIFT", "VULN_DISCLOSURE",
	"ACCOUNT_COMPROMISE", "INSIDER_TIP", "RECRUITMENT_CALL", "NONE",
}

// maxClassifyInputChars bounds cost per call — threads can run to
// hundreds of posts, and this is a paid API call per thread, not a free
// regex pass.
const maxClassifyInputChars = 6000

// LLMClassifier sorts a thread's new/changed posts into one of a fixed
// set of signal categories using an LLM, one call per thread. Unlike
// ArtifactDetector this is a judgment call, not a pattern match, and it
// costs money per call — callers should only include it in Detectors
// when explicitly configured (an API key present).
type LLMClassifier struct {
	client openai.Client
	model  string
}

// NewLLMClassifier returns a classifier against apiKey. model is
// typically "gpt-5.5".
func NewLLMClassifier(apiKey, model string) *LLMClassifier {
	return &LLMClassifier{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// Name implements Detector.
func (c *LLMClassifier) Name() string { return "llm_classify" }

type classifyResult struct {
	Class      string  `json:"class"`
	Headline   string  `json:"headline"`
	Rationale  string  `json:"rationale"`
	Confidence float32 `json:"confidence"`
}

var classifySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"class": map[string]any{
			"type": "string",
			"enum": classifications,
		},
		"headline": map[string]any{
			"type": "string",
		},
		"rationale": map[string]any{
			"type": "string",
		},
		"confidence": map[string]any{
			"type": "number",
		},
	},
	"required":             []string{"class", "headline", "rationale", "confidence"},
	"additionalProperties": false,
}

const classifyPrompt = `You are monitoring 4chan threads for specific signal categories, as part of an early-detection system for tech/AI leaks and industry news. Given a thread's subject and posts below, classify it into exactly one of:

ARTIFACT_DROP: a concrete artifact (model weights, code, dataset) is being shared or linked.
CAPABILITY_CLAIM: someone claims a new capability exists for a model, tool, or system.
LEAK_DOC: a leaked document (internal memo, screenshot of internal comms, contract) is shared.
TOOLING: a new tool, script, or utility is being shared or announced.
MISUSE_DEMAND: someone is asking for or demanding a way to misuse a tool/model (jailbreak, exploit, uncensored access).
CORP_NEWS: corporate news (layoffs, funding, executive departures, policy changes) discussed before official announcement.
SENTIMENT_SHIFT: a notable shift in community sentiment about a company, product, or model.
VULN_DISCLOSURE: a security vulnerability or exploit is discussed, especially before public disclosure.
ACCOUNT_COMPROMISE: credential dumps or breach data are being shared or discussed.
INSIDER_TIP: an unverified claim of firsthand insider knowledge, with no document attached.
RECRUITMENT_CALL: a call to organize, recruit, or coordinate a movement or campaign.
NONE: none of the above clearly apply — use this for ordinary discussion.

Most threads are NONE. Only pick another category when the thread clearly matches it.

For threads that aren't NONE, also produce:

- headline: one plain sentence, written for someone who has not read the thread. State what was found and why it might matter. Do not restate the category name or use jargon (e.g. never "an artifact drop was detected" — say what the artifact actually is).
- rationale: 2-3 sentences on what specifically triggered the flag — name the signal (a concrete pattern, how specific the claim is, whether other posters corroborated it) — and say plainly what is NOT being claimed or verified (e.g. "this is one anon's unverified claim" or "no confirmation from other posters yet").
- confidence: a number from 0 to 1 for how strongly the evidence in *this thread* supports the classification — not how interesting the thread is. Reserve above 0.8 for a concrete, checkable artifact or claim; use 0.3-0.5 for a single uncorroborated claim with nothing to verify it.

For NONE, still fill headline/rationale/confidence with any placeholder text — they're discarded.`

// Detect implements Detector. It makes one LLM call per thread — posts
// is the in-memory set of new/changed posts for this cycle, and nothing
// derived from it is retained beyond this call other than the model's
// own one-sentence rationale.
func (c *LLMClassifier) Detect(board string, thread fourchan.Thread, posts []fourchan.Post) []Finding {
	input := buildClassifyInput(thread, posts)

	resp, err := c.client.Responses.New(context.Background(), responses.ResponseNewParams{
		Model: c.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(classifyPrompt + "\n\n" + input),
		},
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigParamOfJSONSchema("thread_classification", classifySchema),
		},
	})
	if err != nil {
		slog.Error("llm classify request failed", "board", board, "thread", thread.No, "error", err)
		return nil
	}

	var result classifyResult
	if err := json.Unmarshal([]byte(resp.OutputText()), &result); err != nil {
		slog.Error("llm classify response unparseable", "board", board, "thread", thread.No, "error", err)
		return nil
	}
	if result.Class == "" {
		slog.Error("llm classify returned no class", "board", board, "thread", thread.No)
		return nil
	}
	if result.Class == "NONE" {
		return nil
	}

	confidence := result.Confidence
	return []Finding{{
		Board:         board,
		ThreadNo:      thread.No,
		PostNo:        thread.No,
		PostTime:      thread.PostTime(),
		Detector:      c.Name(),
		Kind:          result.Class,
		Note:          result.Rationale,
		ThreadSubject: thread.Sub,
		ThreadReplies: thread.Replies,
		Headline:      result.Headline,
		Rationale:     result.Rationale,
		Confidence:    &confidence,
		Model:         c.model,
	}}
}

func buildClassifyInput(thread fourchan.Thread, posts []fourchan.Post) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Thread subject: %s\n\n", thread.Sub)
	for _, p := range posts {
		text := plainText(p.Com)
		if text == "" {
			continue
		}
		fmt.Fprintf(&b, "Post %d: %s\n", p.No, text)
		if b.Len() > maxClassifyInputChars {
			break
		}
	}
	out := b.String()
	if len(out) > maxClassifyInputChars {
		out = out[:maxClassifyInputChars]
	}
	return out
}
