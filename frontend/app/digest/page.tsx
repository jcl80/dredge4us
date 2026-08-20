import { Navbar } from "../nav";

// Placeholder — the real digest (README §Screens/3) lands in a later commit
// once the summarizer emits picks per window.
export default function DigestPage() {
  return (
    <div className="flex-1 bg-paper">
      <Navbar active="digest" />
      <main className="mx-auto max-w-[1360px] px-5 py-10">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-ink">Digest</h1>
        <p className="mt-2 max-w-[62ch] text-[13.5px] text-ink3">
          The same findings at three maturities: the hour is raw, the day has been checked for
          corroboration, the week carries a verdict.
        </p>
      </main>
    </div>
  );
}
