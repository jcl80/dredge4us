import Link from "next/link";
import { GeneralLineage, getBoards, getGenerals, threadURL } from "../findings";
import { Navbar } from "../nav";
import { boardChipClass, kindBadgeChipClass, timeAgo } from "../ui";

// tickColor matches the density thresholds in README §Screens/4.
function tickColor(count: number): string {
  if (count >= 3) return "#1f7a5a";
  if (count === 2) return "rgba(31,122,90,0.55)";
  if (count === 1) return "rgba(31,122,90,0.28)";
  return "rgba(26,23,20,0.06)";
}

function GeneralCard({ g }: { g: GeneralLineage }) {
  const live = !g.endedAt;
  const firstSeen = new Date(g.firstSeenAt).toLocaleDateString();

  return (
    <div className="flex flex-col gap-2.5 rounded-lg border border-ink/10 bg-panel px-4.5 py-3.5">
      <div className="flex flex-wrap items-center gap-2.5">
        <span className={`size-[7px] rounded-full ${live ? "bg-accent" : "bg-ink4"}`} />
        <a
          href={threadURL(g.board, g.threadNo)}
          target="_blank"
          rel="noopener noreferrer"
          className="text-[14.5px] font-medium text-ink hover:underline"
        >
          {g.threadSubject || `#${g.threadNo}`}
        </a>
        <span className="font-mono text-[11px] text-ink3">/{g.board}/</span>
        <span className={`font-mono text-[11px] ${live ? "text-accent" : "text-ink4"}`}>
          {live ? "live" : "ended"}
        </span>
        <span className="ml-auto font-mono text-[11px] text-ink4">last {timeAgo(g.lastSeenAt)}</span>
      </div>

      <div className="flex gap-[3px]">
        {(g.instanceDensity ?? []).map((count, i) => (
          <span
            key={i}
            title={`${count} finding${count === 1 ? "" : "s"}`}
            className="h-4 flex-1 rounded-sm"
            style={{ backgroundColor: tickColor(count) }}
          />
        ))}
      </div>

      <div className="flex flex-wrap justify-between gap-x-3 font-mono text-[10px] text-ink4">
        <span>{firstSeen}</span>
        <span>
          {g.instanceCount} instance{g.instanceCount === 1 ? "" : "s"} since {firstSeen}
        </span>
        <span>now</span>
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        {g.findingKinds.map((k) => (
          <span key={k} className={kindBadgeChipClass(k)}>
            {k}
          </span>
        ))}
        <span className="ml-auto text-[12px] text-ink3">{g.replies} replies in the live instance</span>
      </div>
    </div>
  );
}

export default async function GeneralsPage(props: PageProps<"/generals">) {
  const searchParams = await props.searchParams;
  const boards = await getBoards();

  const requested = typeof searchParams.board === "string" ? searchParams.board : undefined;
  const board = requested ?? boards[0];
  const generals = board ? await getGenerals(board) : [];

  return (
    <div className="flex-1 bg-paper">
      <Navbar active="generals" board={board} />
      <main className="mx-auto max-w-[1100px] px-5 pt-5 pb-16">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-ink">Generals</h1>
        <p className="mt-2 max-w-[66ch] text-[13.5px] text-ink3">
          Recurring threads stitched across reposts. Each tick is one instance the poller saw; the
          lineage survives after 4chan prunes the original.
        </p>

        <div className="mt-4 flex flex-wrap gap-[5px]">
          {boards.map((b) => (
            <Link key={b} href={`/generals?board=${b}`} className={boardChipClass(board === b)}>
              /{b}/
            </Link>
          ))}
        </div>

        <div className="mt-5 flex flex-col gap-3">
          {generals.map((g) => (
            <GeneralCard key={g.subjectKey} g={g} />
          ))}
          {generals.length === 0 && (
            <div className="py-10 text-center text-[13px] text-ink4">
              No generals tracked yet{board ? ` for /${board}/` : ""}.
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
