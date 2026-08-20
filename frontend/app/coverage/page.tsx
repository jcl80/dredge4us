import { Navbar } from "../nav";

// Placeholder — the real coverage screen (README §Screens/5) lands once the
// /coverage aggregate endpoint (commit 11) replaces the old per-board N+1.
export default function CoveragePage() {
  return (
    <div className="flex-1 bg-paper">
      <Navbar active="coverage" />
      <main className="mx-auto max-w-[1100px] px-5 py-10">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-ink">Coverage</h1>
        <p className="mt-2 max-w-[66ch] text-[13.5px] text-ink3">
          What the poller watches, what it costs, and what to add next. Yield is findings kept
          per thousand posts read — the number that decides whether a board earns a slot.
        </p>
      </main>
    </div>
  );
}
