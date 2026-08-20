"use client";

import { useState } from "react";
import type { NarrativeSummary, SummaryWindow } from "../findings";
import { kindAccentColor, timeAgo } from "../ui";

type WindowKey = NarrativeSummary["window"];

const TABS: { key: WindowKey; label: string }[] = [
  { key: "hour", label: "Last hour" },
  { key: "day", label: "Last 24 hours" },
  { key: "week", label: "Last 7 days" },
];

function tabClass(active: boolean): string {
  return active
    ? "border-b-2 border-accent px-3 py-2 text-[13px] font-semibold text-ink"
    : "border-b-2 border-transparent px-3 py-2 text-[13px] text-ink3 hover:text-ink";
}

// splitLede takes the summarizer's single prose string and pulls its
// first sentence out as the lede, styled larger — README's spec wants a
// lede/body split, but the summarizer only emits one string, so this is
// a presentational split of real generated text, not fabricated content.
function splitLede(summary: string): [string, string] {
  const match = summary.match(/^(.*?[.!?])(\s+)([\s\S]*)$/);
  if (!match) return [summary, ""];
  return [match[1], match[3]];
}

export function DigestView({
  narratives,
  summaryWindows,
  boardCount,
}: {
  narratives: NarrativeSummary[];
  summaryWindows: SummaryWindow[];
  boardCount: number;
}) {
  const [activeWindow, setActiveWindow] = useState<WindowKey>("hour");

  const tab = TABS.find((t) => t.key === activeWindow)!;
  const narrative = narratives.find((n) => n.window === activeWindow) ?? null;
  const summary = summaryWindows.find((w) => w.label === tab.label) ?? null;
  const maxCount = summary ? Math.max(0, ...summary.byKind.map((kc) => kc.count)) : 0;
  const [lede, body] = narrative ? splitLede(narrative.summary) : ["", ""];

  return (
    <main className="mx-auto max-w-[820px] px-5 pt-7 pb-18">
      <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-ink">Digest</h1>
      <p className="mt-2 max-w-[62ch] text-[13.5px] text-ink3">
        The same findings at three maturities: the hour is raw, the day has been checked for
        corroboration, the week carries a verdict.
      </p>

      <div className="mt-5 flex flex-wrap items-center border-b border-ink/10">
        {TABS.map((t) => (
          <button key={t.key} onClick={() => setActiveWindow(t.key)} className={tabClass(activeWindow === t.key)}>
            {t.label}
          </button>
        ))}
        {narrative && (
          <span className="ml-auto font-mono text-[11px] text-ink4">
            generated {timeAgo(narrative.generatedAt)}
          </span>
        )}
      </div>

      <div className="mt-3 font-mono text-[11px] text-ink4">
        {tab.label} · {summary?.totalFindings ?? 0} finding{summary?.totalFindings === 1 ? "" : "s"} ·{" "}
        {boardCount} boards
      </div>

      {narrative ? (
        <>
          <p className="mt-5 max-w-[64ch] text-[17px] leading-[1.62] text-[#26221e]">{lede}</p>
          {body && <p className="mt-3 max-w-[64ch] text-[15px] leading-[1.68] text-ink2">{body}</p>}
        </>
      ) : (
        <p className="mt-5 text-[13.5px] text-ink4">Narrative not generated yet for this window.</p>
      )}

      {/* "Worth clicking" needs a `picks` field the summarizer doesn't
          produce yet — shipping without it per the plan's own fallback
          rather than faking verdicts. */}

      {summary && summary.byKind.length > 0 && (
        <div className="mt-8">
          <div className="font-mono text-[10.5px] tracking-[0.06em] text-ink4 uppercase">Volume by kind</div>
          <div className="mt-2.5 flex flex-col gap-2">
            {summary.byKind.map((kc) => (
              <div key={kc.kind} className="grid grid-cols-[170px_1fr_34px] items-center gap-3">
                <span className="truncate font-mono text-[10.5px] text-ink3">{kc.kind}</span>
                <span className="block h-2 overflow-hidden rounded-sm bg-ink/[.06]">
                  <span
                    className="block h-full rounded-sm"
                    style={{
                      width: `${maxCount > 0 ? (kc.count / maxCount) * 100 : 0}%`,
                      backgroundColor: kindAccentColor(kc.kind),
                    }}
                  />
                </span>
                <span className="text-right font-mono text-[11px] text-ink3">{kc.count}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </main>
  );
}
