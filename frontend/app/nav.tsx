import Link from "next/link";

type NavLink = "findings" | "generals" | "boards";

function linkClass(active: boolean): string {
  return active
    ? "font-medium text-black dark:text-zinc-50"
    : "text-zinc-600 hover:text-black dark:text-zinc-400 dark:hover:text-zinc-50";
}

// Navbar is rendered per-page (not in layout.tsx) so it can carry the
// current board filter across Findings <-> Generals — layouts don't
// receive the searchParams prop in this Next.js version.
export function Navbar({ active, board }: { active: NavLink; board?: string }) {
  const suffix = board ? `?board=${board}` : "";

  return (
    <header className="border-b border-black/[.08] bg-white dark:border-white/[.145] dark:bg-black">
      <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
        <Link href="/" className="font-semibold text-black dark:text-zinc-50">
          dredge4us
        </Link>
        <nav className="flex gap-5 text-sm">
          <Link href={`/${suffix}`} className={linkClass(active === "findings")}>
            Findings
          </Link>
          <Link href={`/generals${suffix}`} className={linkClass(active === "generals")}>
            Generals
          </Link>
          <Link href="/boards" className={linkClass(active === "boards")}>
            Boards
          </Link>
        </nav>
      </div>
    </header>
  );
}
