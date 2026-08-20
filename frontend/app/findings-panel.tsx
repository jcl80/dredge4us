"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { archiveNote, Finding, FindingContext, sourceLabel, threadURL } from "./findings";
import { confidenceInk, confidencePct, detectorLabel, kindBadgeChipClass, timeAgo } from "./ui";

// headlineFor falls back to matchedValue, then note, when the classifier
// hasn't populated headline yet (regex findings never do).
function headlineFor(f: Finding): string {
  return f.headline || f.matchedValue || f.note || "";
}

function matchesQuery(f: Finding, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  return (
    headlineFor(f).toLowerCase().includes(q) ||
    f.kind.toLowerCase().includes(q) ||
    f.matchedValue.toLowerCase().includes(q) ||
    f.threadSubject.toLowerCase().includes(q)
  );
}

const GROUP_ORDER = ["Last hour", "Earlier today", "Yesterday", "Older"] as const;
type GroupLabel = (typeof GROUP_ORDER)[number];

// groupByRecency buckets into the three windows the design calls for,
// plus "Older" (not in the design — the design assumes everything fits
// in the last two days; this is a fallback so nothing silently
// disappears from the list once it doesn't).
function groupByRecency(findings: Finding[]): { label: GroupLabel; items: Finding[] }[] {
  const now = Date.now();
  const lastHour = now - 60 * 60 * 1000;
  const todayStart = new Date();
  todayStart.setHours(0, 0, 0, 0);
  const yesterdayStart = new Date(todayStart);
  yesterdayStart.setDate(yesterdayStart.getDate() - 1);

  const buckets: Record<GroupLabel, Finding[]> = {
    "Last hour": [],
    "Earlier today": [],
    Yesterday: [],
    Older: [],
  };

  for (const f of findings) {
    const t = new Date(f.foundAt).getTime();
    if (t >= lastHour) buckets["Last hour"].push(f);
    else if (t >= todayStart.getTime()) buckets["Earlier today"].push(f);
    else if (t >= yesterdayStart.getTime()) buckets.Yesterday.push(f);
    else buckets.Older.push(f);
  }

  return GROUP_ORDER.filter((label) => buckets[label].length > 0).map((label) => ({
    label,
    items: buckets[label],
  }));
}

function boardChipClass(active: boolean): string {
  return `rounded-full border px-2.5 py-[3px] font-mono text-[11px] ${
    active ? "border-ink bg-ink text-white" : "border-ink/[.12] bg-panel text-ink2"
  }`;
}

export function FindingsWorkspace({
  findings,
  boards,
  board,
}: {
  findings: Finding[];
  boards: string[];
  board?: string;
}) {
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<number | null>(findings[0]?.id ?? null);
  const [copied, setCopied] = useState(false);
  const [context, setContext] = useState<FindingContext | null>(null);

  const filtered = findings.filter((f) => matchesQuery(f, query));
  const groups = groupByRecency(filtered);
  const selected = findings.find((f) => f.id === selectedId) ?? findings[0] ?? null;

  function select(f: Finding) {
    setSelectedId(f.id);
    setCopied(false);
  }

  // Deliberately does not reset context to null on selection change —
  // holding the previous finding's content while the new one loads
  // avoids the panel collapsing height and popping back, per the design
  // spec. A stale-response guard covers fast repeated selection.
  const effectiveSelectedId = selected?.id ?? null;
  useEffect(() => {
    if (effectiveSelectedId === null) return;
    let stale = false;
    fetch(`/api/findings/${effectiveSelectedId}/context`)
      .then((res) => (res.ok ? (res.json() as Promise<FindingContext>) : null))
      .then((data) => {
        if (!stale && data) setContext(data);
      })
      .catch(() => {
        // Keep showing whatever context is already there.
      });
    return () => {
      stale = true;
    };
  }, [effectiveSelectedId]);

  return (
    <main className="mx-auto flex max-w-[1360px] flex-wrap items-start gap-5 px-5 pt-5 pb-16">
      <section className="min-w-[340px] flex-1 basis-3/4 overflow-hidden rounded-lg border border-ink/10 bg-panel">
        <div className="border-b border-ink/[.08] px-4 pt-3.5 pb-3">
          <div className="flex flex-wrap items-baseline gap-2.5">
            <h1 className="text-base font-semibold tracking-[-0.01em] text-ink">Findings</h1>
            <span className="font-mono text-[11px] text-ink4">
              {filtered.length} of {findings.length} · sorted by recency
            </span>
          </div>
          <p className="mt-1.5 max-w-[46ch] text-pretty text-[12.5px] text-ink3">
            Posts a detector thought were worth a second look. Every row is one post; the panel
            explains why it was flagged.
          </p>

          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter findings, artifacts, threads"
            className="mt-3 w-full rounded-md border border-ink/[.14] bg-[#fdfcfb] px-2.5 py-[7px] text-[13px] text-ink outline-none placeholder:text-ink4"
          />

          <div className="mt-2.5 flex flex-wrap gap-[5px]">
            <Link href="/" className={boardChipClass(!board)}>
              all boards
            </Link>
            {boards.map((b) => (
              <Link key={b} href={`/?board=${b}`} className={boardChipClass(board === b)}>
                /{b}/
              </Link>
            ))}
          </div>
        </div>

        <div className="max-h-[74vh] overflow-y-auto">
          {groups.map((g) => (
            <div key={g.label}>
              <div className="sticky top-0 border-b border-ink/[.07] bg-sunk px-4 py-[5px] font-mono text-[10.5px] tracking-[0.06em] text-ink4 uppercase">
                {g.label}
              </div>
              {g.items.map((f) => (
                <div
                  key={f.id}
                  onClick={() => select(f)}
                  className={`grid cursor-pointer grid-cols-[46px_1fr] gap-3 border-b border-ink/[.06] px-4 pt-[11px] pb-3 ${
                    selected?.id === f.id ? "bg-[#f4f7f5] shadow-[inset_3px_0_0_var(--color-accent)]" : ""
                  }`}
                >
                  <div className="pt-px font-mono text-[11px] text-ink4">
                    <div>{timeAgo(f.foundAt)}</div>
                    <div className="mt-[3px] text-ink3">/{f.board}/</div>
                  </div>
                  <div className="min-w-0">
                    <div className="text-pretty text-[13.5px] font-medium text-ink">{headlineFor(f)}</div>
                    <div className="mt-[7px] flex flex-wrap items-center gap-1.5">
                      <span className={kindBadgeChipClass(f.kind)}>{f.kind}</span>
                      {f.confidence != null ? (
                        <span className="ml-auto flex items-center gap-[5px]">
                          <span className="block h-[3px] w-11 overflow-hidden rounded-full bg-ink/10">
                            <span
                              className="block h-full rounded-full"
                              style={{
                                width: confidencePct(f.confidence),
                                backgroundColor: confidenceInk(f.confidence),
                              }}
                            />
                          </span>
                          <span className="font-mono text-[10px] text-ink4">{confidencePct(f.confidence)}</span>
                        </span>
                      ) : (
                        <span className="ml-auto font-mono text-[10px] text-ink4">—</span>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ))}
          {filtered.length === 0 && (
            <div className="py-10 text-center text-[13px] text-ink4">Nothing matches that filter.</div>
          )}
        </div>
      </section>

      <section className="sticky top-24 min-w-[340px] flex-1 basis-1/4">
        <div className="overflow-hidden rounded-lg border border-ink/10 bg-panel">
          {selected ? (
            <>
              <div className="border-b border-ink/[.08] px-5.5 pt-6 pb-4">
                <div className="flex flex-wrap items-center gap-2 font-mono text-[11px] text-ink4">
                  <span className={kindBadgeChipClass(selected.kind)}>{selected.kind}</span>
                  <span>/{selected.board}/</span>
                  <span>·</span>
                  <span>post {selected.postNo}</span>
                  <span>·</span>
                  <span>{timeAgo(selected.foundAt)}</span>
                  <span>·</span>
                  <span>{detectorLabel(selected.detector)}</span>
                </div>
                <h2 className="mt-2.5 text-pretty text-[20px] leading-[1.3] font-semibold tracking-[-0.015em] text-ink">
                  {headlineFor(selected)}
                </h2>
              </div>

              {selected.rationale && (
                <div className="border-b border-ink/[.08] bg-[#fbfaf8] px-5.5 py-4">
                  <div className="font-mono text-[10.5px] tracking-[0.06em] text-ink4 uppercase">
                    Why it was flagged
                  </div>
                  <p className="mt-2 max-w-[68ch] text-pretty text-[13.5px] text-ink2">{selected.rationale}</p>
                  <div className="mt-3 flex flex-wrap gap-3.5 font-mono text-[11px] text-ink3">
                    {selected.confidence != null && <span>confidence {confidencePct(selected.confidence)}</span>}
                    {selected.rule && <span>rule {selected.rule}</span>}
                    {selected.model && <span>model {selected.model}</span>}
                  </div>
                </div>
              )}

              {selected.matchedValue && (
                <div className="border-b border-ink/[.08] px-5.5 py-4">
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-mono text-[10.5px] tracking-[0.06em] text-ink4 uppercase">
                      Matched artifact
                    </div>
                    <button
                      onClick={() => {
                        void navigator.clipboard.writeText(selected.matchedValue);
                        setCopied(true);
                      }}
                      className="rounded-[5px] border border-ink/[.14] px-2 py-[3px] text-[11px] text-ink2"
                    >
                      {copied ? "copied" : "copy"}
                    </button>
                  </div>
                  <div className="mt-2 rounded-md border border-ink/10 bg-sunk px-3 py-2.5 font-mono text-[12px] text-ink [overflow-wrap:anywhere]">
                    {selected.matchedValue}
                  </div>
                </div>
              )}

              {context && (
                <div className="border-b border-ink/[.08] px-5.5 py-4">
                  <div className="font-mono text-[10.5px] tracking-[0.06em] text-ink4 uppercase">Post</div>
                  {context.postText && (
                    <blockquote className="mt-2 max-w-[68ch] border-l-2 border-[rgba(31,122,90,0.35)] pl-3 text-[13.5px] whitespace-pre-wrap text-ink2">
                      {context.postText}
                    </blockquote>
                  )}
                  <div className="mt-3 text-[12.5px] text-ink3">
                    In{" "}
                    <a
                      href={threadURL(selected.board, selected.threadNo)}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:underline"
                    >
                      {selected.threadSubject || `#${selected.threadNo}`}
                    </a>{" "}
                    · {selected.threadReplies} replies · {sourceLabel(selected.board)}
                  </div>
                </div>
              )}

              {context && (
                <div className="border-b border-ink/[.08] px-5.5 py-4">
                  <div className="font-mono text-[10.5px] tracking-[0.06em] text-ink4 uppercase">
                    Else in this thread
                  </div>
                  <div className="mt-2 flex flex-col gap-2">
                    {context.neighbors.map((n) => (
                      <div key={n.id} className="flex items-baseline gap-2.5 text-[12.5px] text-ink2">
                        <span className={`${kindBadgeChipClass(n.kind)} whitespace-nowrap`}>{n.kind}</span>
                        <span className="text-pretty">{headlineFor(n)}</span>
                        <span className="ml-auto font-mono text-[10.5px] whitespace-nowrap text-ink4">
                          {timeAgo(n.foundAt)}
                        </span>
                      </div>
                    ))}
                    {context.neighbors.length === 0 && (
                      <div className="text-[12.5px] text-ink4">Nothing else flagged in this thread.</div>
                    )}
                  </div>
                </div>
              )}

              <div className="flex flex-wrap items-center gap-2 px-5.5 py-3.5">
                <a
                  href={threadURL(selected.board, selected.threadNo, selected.postNo)}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="rounded-md bg-accent px-3 py-[7px] text-[12.5px] font-medium text-white"
                >
                  Open source post
                </a>
                {/* Wired in a later commit: POST /findings/{id}/noise + post-click state. */}
                <button className="rounded-md border border-ink/[.14] bg-panel px-3 py-[7px] text-[12.5px] text-ink2">
                  Mark as noise
                </button>
                <span className="ml-auto font-mono text-[11px] text-ink4">{archiveNote(selected.board)}</span>
              </div>
            </>
          ) : (
            <div className="px-5.5 py-8 text-center text-[13px] text-ink4">
              Select a finding to see why it was flagged.
            </div>
          )}
        </div>

        <p className="mt-2.5 max-w-[60ch] text-pretty text-[11.5px] text-ink4">
          Marking noise feeds the precision counter the roadmap calls for — flags are what let you
          measure the classifier instead of guessing at it.
        </p>
      </section>
    </main>
  );
}
