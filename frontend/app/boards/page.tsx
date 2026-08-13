import Link from "next/link";
import { getBoards, getGenerals, getKinds } from "../findings";
import { Navbar } from "../nav";

type BoardRow = {
  board: string;
  liveGenerals: number;
  totalGenerals: number;
  kinds: number;
};

export default async function BoardsPage() {
  const boards = await getBoards();

  const rows: BoardRow[] = await Promise.all(
    boards.map(async (board) => {
      const [generals, kinds] = await Promise.all([getGenerals(board), getKinds(board)]);
      return {
        board,
        liveGenerals: generals.filter((g) => !g.endedAt).length,
        totalGenerals: generals.length,
        kinds: kinds.length,
      };
    }),
  );

  return (
    <div className="flex-1 bg-zinc-50 dark:bg-black">
      <Navbar active="boards" />
      <main className="mx-auto max-w-5xl px-6 py-10">
        <h1 className="text-2xl font-semibold text-black dark:text-zinc-50">Boards</h1>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Boards currently watched by the poller.
        </p>

        <div className="mt-6 overflow-x-auto rounded-lg border border-black/[.08] dark:border-white/[.145]">
          <table className="w-full text-left text-sm">
            <thead className="bg-black/[.03] dark:bg-white/[.04]">
              <tr>
                <th className="px-4 py-2 font-medium">Board</th>
                <th className="px-4 py-2 font-medium">Live generals</th>
                <th className="px-4 py-2 font-medium">Generals tracked</th>
                <th className="px-4 py-2 font-medium">Finding kinds</th>
                <th className="px-4 py-2 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.board} className="border-t border-black/[.08] dark:border-white/[.145]">
                  <td className="px-4 py-2 font-medium text-black dark:text-zinc-50">
                    /{r.board}/
                  </td>
                  <td className="px-4 py-2">{r.liveGenerals}</td>
                  <td className="px-4 py-2">{r.totalGenerals}</td>
                  <td className="px-4 py-2">{r.kinds}</td>
                  <td className="px-4 py-2 whitespace-nowrap text-right">
                    <Link
                      href={`/?board=${r.board}`}
                      className="text-zinc-600 hover:underline dark:text-zinc-400"
                    >
                      Findings
                    </Link>
                    <span className="mx-2 text-zinc-400">&middot;</span>
                    <Link
                      href={`/generals?board=${r.board}`}
                      className="text-zinc-600 hover:underline dark:text-zinc-400"
                    >
                      Generals
                    </Link>
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-zinc-500">
                    No boards configured.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </main>
    </div>
  );
}
