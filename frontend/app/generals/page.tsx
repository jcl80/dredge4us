import Link from "next/link";
import { getBoards, getGenerals } from "../findings";
import { kindBadgeClass, pillClass, timeAgo } from "../ui";

export default async function GeneralsPage(props: PageProps<"/generals">) {
  const searchParams = await props.searchParams;
  const boards = await getBoards();

  const requested = typeof searchParams.board === "string" ? searchParams.board : undefined;
  const board = requested ?? boards[0];
  const generals = board ? await getGenerals(board) : [];

  return (
    <div className="flex-1 bg-zinc-50 dark:bg-black">
      <main className="mx-auto max-w-5xl px-6 py-10">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-semibold text-black dark:text-zinc-50">
            Generals
          </h1>
          <Link href="/" className="text-sm text-zinc-600 hover:underline dark:text-zinc-400">
            &larr; Findings
          </Link>
        </div>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Recurring general threads tracked{board ? ` for /${board}/` : ""}.
        </p>

        <nav className="mt-4 flex flex-wrap gap-2 text-sm">
          {boards.map((b) => (
            <Link key={b} href={`/generals?board=${b}`} className={pillClass(board === b)}>
              /{b}/
            </Link>
          ))}
        </nav>

        <div className="mt-6 overflow-x-auto rounded-lg border border-black/[.08] dark:border-white/[.145]">
          <table className="w-full text-left text-sm">
            <thead className="bg-black/[.03] dark:bg-white/[.04]">
              <tr>
                <th className="px-4 py-2 font-medium">Subject</th>
                <th className="px-4 py-2 font-medium">Findings</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Replies</th>
                <th className="px-4 py-2 font-medium">Instances</th>
                <th className="px-4 py-2 font-medium">Last active</th>
              </tr>
            </thead>
            <tbody>
              {generals.map((g) => (
                <tr
                  key={g.subjectKey}
                  className="border-t border-black/[.08] dark:border-white/[.145]"
                >
                  <td className="max-w-sm truncate px-4 py-2">
                    <a
                      href={`https://boards.4chan.org/${g.board}/thread/${g.threadNo}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:underline"
                    >
                      {g.threadSubject}
                    </a>
                  </td>
                  <td className="px-4 py-2">
                    {g.findingKinds.length === 0 ? (
                      <span className="text-zinc-400">&mdash;</span>
                    ) : (
                      <div className="flex flex-wrap gap-1">
                        {g.findingKinds.map((k) => (
                          <span
                            key={k}
                            className={`rounded px-2 py-0.5 text-xs font-medium ${kindBadgeClass(k)}`}
                          >
                            {k}
                          </span>
                        ))}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-2 whitespace-nowrap">
                    {g.endedAt ? (
                      <span className="text-zinc-500">ended</span>
                    ) : (
                      <span className="text-green-600 dark:text-green-400">live</span>
                    )}
                  </td>
                  <td className="px-4 py-2 whitespace-nowrap">{g.replies}</td>
                  <td className="px-4 py-2 whitespace-nowrap">
                    {g.instanceCount} thread{g.instanceCount === 1 ? "" : "s"} since{" "}
                    {new Date(g.firstSeenAt).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-2 whitespace-nowrap text-zinc-600 dark:text-zinc-400">
                    {timeAgo(g.lastSeenAt)}
                  </td>
                </tr>
              ))}
              {generals.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-zinc-500">
                    No generals tracked yet{board ? ` for /${board}/` : ""}.
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
