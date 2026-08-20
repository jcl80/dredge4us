import { DigestView } from "./digest-view";
import { getBoards, getNarrativeSummaries, getSummary } from "../findings";
import { Navbar } from "../nav";

export default async function DigestPage() {
  const [boards, narratives, summaryWindows] = await Promise.all([
    getBoards(),
    getNarrativeSummaries(),
    getSummary(),
  ]);

  return (
    <div className="flex-1 bg-paper">
      <Navbar active="digest" />
      <DigestView narratives={narratives} summaryWindows={summaryWindows} boardCount={boards.length} />
    </div>
  );
}
