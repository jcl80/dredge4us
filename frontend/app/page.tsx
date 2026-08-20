import { FindingsWorkspace } from "./findings-panel";
import { getBoards, getFindings } from "./findings";
import { Navbar } from "./nav";

export default async function Home(props: PageProps<"/">) {
  const searchParams = await props.searchParams;
  const board = typeof searchParams.board === "string" ? searchParams.board : undefined;

  const [boards, findings] = await Promise.all([getBoards(), getFindings(board)]);

  return (
    <div className="flex-1 bg-paper">
      <Navbar active="findings" board={board} />
      <FindingsWorkspace findings={findings} boards={boards} board={board} />
    </div>
  );
}
