import Link from "next/link";
import { getBoards } from "./findings";

type NavLink = "findings" | "digest" | "generals" | "coverage";

function tabClass(active: boolean): string {
  return active
    ? "border-b-2 border-accent px-3 py-2 text-[13px] font-semibold text-ink"
    : "border-b-2 border-transparent px-3 py-2 text-[13px] text-ink3 hover:text-ink";
}

// Navbar is rendered per-page (not in layout.tsx) so it can carry the
// current board filter across Findings <-> Generals — layouts don't
// receive the searchParams prop in this Next.js version.
export async function Navbar({ active, board }: { active: NavLink; board?: string }) {
  const suffix = board ? `?board=${board}` : "";
  const boards = await getBoards();

  return (
    <header className="sticky top-0 z-20 border-b border-ink/10 bg-paper/90 backdrop-blur">
      <div className="mx-auto flex min-h-14 max-w-[1360px] flex-wrap items-center gap-6 px-5 py-2.5">
        <div className="flex items-baseline gap-2.5">
          <Link href="/" className="font-mono text-[15px] font-semibold tracking-[-0.01em] text-ink">
            dredge4us
          </Link>
          <span className="text-[11px] text-ink4">4chan signal dredge</span>
        </div>

        <nav className="order-3 flex basis-full gap-0.5 border-t border-ink/[.07] pt-0.5">
          <Link href={`/${suffix}`} className={tabClass(active === "findings")}>
            Findings
          </Link>
          <Link href="/digest" className={tabClass(active === "digest")}>
            Digest
          </Link>
          <Link href={`/generals${suffix}`} className={tabClass(active === "generals")}>
            Generals
          </Link>
          <Link href="/coverage" className={tabClass(active === "coverage")}>
            Coverage
          </Link>
        </nav>

        <div className="ml-auto flex items-center gap-2 rounded-full border border-ink/10 bg-panel py-1 pl-2 pr-2.5">
          <span className="size-1.5 animate-dredge-pulse rounded-full bg-accent" />
          {/* TODO: append "· last {n}s ago" once a poller heartbeat is exposed */}
          <span className="font-mono text-[11px] text-ink3">polling {boards.length} boards</span>
        </div>
      </div>
    </header>
  );
}
