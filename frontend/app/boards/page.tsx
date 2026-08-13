import Link from "next/link";
import { getAllBoards, getGenerals, getKinds } from "../findings";
import { Navbar } from "../nav";

type BoardRow = {
  board: string;
  title: string;
  watched: boolean;
  liveGenerals: number;
  totalGenerals: number;
  kinds: number;
};

// Stats come from our own DB, not just watched boards' live polling — a
// board can have generals data from a one-off catalog snapshot without
// being in POLLER_BOARDS, so every board is queried the same way.
export default async function BoardsPage() {
  const all = await getAllBoards();

  const rows: BoardRow[] = await Promise.all(
    all.map(async (b) => {
      const [generals, kinds] = await Promise.all([getGenerals(b.board), getKinds(b.board)]);
      return {
        ...b,
        liveGenerals: generals.filter((g) => !g.endedAt).length,
        totalGenerals: generals.length,
        kinds: kinds.length,
      };
    }),
  );

  rows.sort(
    (a, b) =>
      Number(b.watched) - Number(a.watched) ||
      b.totalGenerals - a.totalGenerals ||
      a.board.localeCompare(b.board),
  );

  return (
    <div className="flex-1 bg-zinc-50 dark:bg-black">
      <Navbar active="boards" />
      <main className="mx-auto max-w-5xl px-6 py-10">
        <h1 className="text-2xl font-semibold text-black dark:text-zinc-50">Boards</h1>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Every board 4chan currently serves, tagged by whether the poller watches it.
        </p>

        <div className="mt-6 overflow-x-auto rounded-lg border border-black/[.08] dark:border-white/[.145]">
          <table className="w-full text-left text-sm">
            <thead className="bg-black/[.03] dark:bg-white/[.04]">
              <tr>
                <th className="px-4 py-2 font-medium">Board</th>
                <th className="px-4 py-2 font-medium">Title</th>
                <th className="px-4 py-2 font-medium">Status</th>
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
                  <td className="px-4 py-2 text-zinc-600 dark:text-zinc-400">{r.title}</td>
                  <td className="px-4 py-2 whitespace-nowrap">
                    {r.watched ? (
                      <span className="text-green-600 dark:text-green-400">watched</span>
                    ) : (
                      <span className="text-zinc-500">unwatched</span>
                    )}
                  </td>
                  <td className="px-4 py-2">
                    {r.liveGenerals || <span className="text-zinc-400">&mdash;</span>}
                  </td>
                  <td className="px-4 py-2">
                    {r.totalGenerals || <span className="text-zinc-400">&mdash;</span>}
                  </td>
                  <td className="px-4 py-2">{r.kinds || <span className="text-zinc-400">&mdash;</span>}</td>
                  <td className="px-4 py-2 whitespace-nowrap text-right">
                    {(r.watched || r.totalGenerals > 0 || r.kinds > 0) && (
                      <>
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
                      </>
                    )}
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-zinc-500">
                    No boards found.
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
