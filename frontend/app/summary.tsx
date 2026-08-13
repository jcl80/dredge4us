import type { NarrativeSummary, SummaryWindow } from "./findings";
import { kindBadgeClass } from "./ui";

const TOP_KINDS = 4;

// Maps SummaryWindow.label (stat cards, board-scoped) to
// NarrativeSummary.window (LLM prose, always all-boards) so the two
// independently-shaped responses can be zipped together per card.
const NARRATIVE_WINDOW_BY_LABEL: Record<string, NarrativeSummary["window"]> = {
  "Last hour": "hour",
  "Last 24 hours": "day",
  "Last 7 days": "week",
};

export function ExecutiveSummary({
  windows,
  narratives,
  scopedToBoard,
}: {
  windows: SummaryWindow[];
  narratives: NarrativeSummary[];
  scopedToBoard: boolean;
}) {
  return (
    <div className="mt-3 grid grid-cols-1 gap-4 sm:grid-cols-3">
      {windows.map((w) => {
        const narrative = narratives.find(
          (n) => n.window === NARRATIVE_WINDOW_BY_LABEL[w.label],
        );

        return (
          <div
            key={w.label}
            className="rounded-lg border border-black/[.08] bg-white px-4 py-3 dark:border-white/[.145] dark:bg-white/[.03]"
          >
            <p className="text-xs font-medium tracking-wide text-zinc-500 uppercase dark:text-zinc-400">
              {w.label}
            </p>

            {narrative ? (
              <p className="mt-2 text-sm text-zinc-700 dark:text-zinc-300">{narrative.summary}</p>
            ) : (
              <p className="mt-2 text-sm text-zinc-500 italic">Narrative not generated yet.</p>
            )}
            {narrative && scopedToBoard && (
              <p className="mt-1 text-xs text-zinc-400 italic">
                Narrative covers all boards, not just this filter.
              </p>
            )}

            <p className="mt-3 text-lg font-semibold text-black dark:text-zinc-50">
              {w.totalFindings}
              <span className="ml-1 text-xs font-normal text-zinc-500 dark:text-zinc-400">
                finding{w.totalFindings === 1 ? "" : "s"}
              </span>
              <span className="ml-2 text-xs font-normal text-zinc-500 dark:text-zinc-400">
                &middot; {w.newGenerals} new general{w.newGenerals === 1 ? "" : "s"}
              </span>
            </p>

            {w.byKind.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1">
                {w.byKind.slice(0, TOP_KINDS).map((kc) => (
                  <span
                    key={kc.kind}
                    className={`rounded px-2 py-0.5 text-xs font-medium ${kindBadgeClass(kc.kind)}`}
                  >
                    {kc.kind} &middot; {kc.count}
                  </span>
                ))}
                {w.byKind.length > TOP_KINDS && (
                  <span className="rounded px-2 py-0.5 text-xs font-medium text-zinc-500">
                    +{w.byKind.length - TOP_KINDS} more
                  </span>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
