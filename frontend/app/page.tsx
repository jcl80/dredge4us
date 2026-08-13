import Link from "next/link";
import { getBoards, getFindings, getKinds } from "./findings";
import { Navbar } from "./nav";
import { kindBadgeClass, pillClass, timeAgo } from "./ui";

function isLinkable(value: string): boolean {
  return value.startsWith("http://") || value.startsWith("https://") || value.startsWith("magnet:");
}

function filterHref(board?: string, kind?: string): string {
  const params = new URLSearchParams();
  if (board) params.set("board", board);
  if (kind) params.set("kind", kind);
  const qs = params.toString();
  return qs ? `/?${qs}` : "/";
}

export default async function Home(props: PageProps<"/">) {
  const searchParams = await props.searchParams;
  const board = typeof searchParams.board === "string" ? searchParams.board : undefined;
  const kind = typeof searchParams.kind === "string" ? searchParams.kind : undefined;

  const [boards, kinds, findings] = await Promise.all([
    getBoards(),
    getKinds(board),
    getFindings(board, kind),
  ]);

  return (
    <div className="flex-1 bg-zinc-50 dark:bg-black">
      <Navbar active="findings" board={board} />
      <main className="mx-auto max-w-5xl px-6 py-10">
        <h1 className="text-2xl font-semibold text-black dark:text-zinc-50">
          Findings
        </h1>
        <div className="mt-3 rounded-lg border border-black/[.08] bg-white px-4 py-3 text-sm text-zinc-700 dark:border-white/[.145] dark:bg-white/[.03] dark:text-zinc-300">
          dredge4us polls watched 4chan boards in real time and flags posts worth a
          second look &mdash; leaked artifacts, capability claims, insider tips, and
          more &mdash; using regex rules and LLM classification. It also tracks
          recurring &quot;general&quot; threads across reposts, so context survives even
          after 4chan prunes the original thread.
        </div>
        <p className="mt-3 text-sm text-zinc-600 dark:text-zinc-400">
          {findings.length} most recent artifact detections.
        </p>

        <nav className="mt-4 flex flex-wrap gap-2 text-sm">
          <Link href={filterHref(undefined, kind)} className={pillClass(!board)}>
            All
          </Link>
          {boards.map((b) => (
            <Link key={b} href={filterHref(b, kind)} className={pillClass(board === b)}>
              /{b}/
            </Link>
          ))}
        </nav>

        <nav className="mt-2 flex flex-wrap gap-2 text-sm">
          <Link href={filterHref(board)} className={pillClass(!kind)}>
            All kinds
          </Link>
          {kinds.map((k) => (
            <Link key={k} href={filterHref(board, k)} className={pillClass(kind === k)}>
              {k}
            </Link>
          ))}
        </nav>

        <div className="mt-6 overflow-x-auto rounded-lg border border-black/[.08] dark:border-white/[.145]">
          <table className="w-full text-left text-sm">
            <thead className="bg-black/[.03] dark:bg-white/[.04]">
              <tr>
                <th className="px-4 py-2 font-medium">Board</th>
                <th className="px-4 py-2 font-medium">Kind</th>
                <th className="px-4 py-2 font-medium">Match / note</th>
                <th className="px-4 py-2 font-medium">Thread</th>
                <th className="px-4 py-2 font-medium">Found</th>
              </tr>
            </thead>
            <tbody>
              {findings.map((f) => (
                <tr
                  key={f.id}
                  className="border-t border-black/[.08] dark:border-white/[.145]"
                >
                  <td className="px-4 py-2 whitespace-nowrap">/{f.board}/</td>
                  <td className="px-4 py-2 whitespace-nowrap">
                    <span className={`rounded px-2 py-0.5 text-xs font-medium ${kindBadgeClass(f.kind)}`}>
                      {f.kind}
                    </span>
                  </td>
                  <td className="max-w-md px-4 py-2 text-xs">
                    {f.matchedValue ? (
                      <span className="block truncate font-mono">
                        {isLinkable(f.matchedValue) ? (
                          <a
                            href={f.matchedValue}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="hover:underline"
                          >
                            {f.matchedValue}
                          </a>
                        ) : (
                          f.matchedValue
                        )}
                      </span>
                    ) : (
                      <span className="italic text-zinc-500">{f.note}</span>
                    )}
                  </td>
                  <td className="max-w-xs px-4 py-2">
                    <a
                      href={`https://boards.4chan.org/${f.board}/thread/${f.threadNo}#p${f.postNo}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:underline"
                    >
                      {f.threadSubject || `#${f.threadNo}`}
                    </a>
                  </td>
                  <td className="px-4 py-2 whitespace-nowrap text-zinc-600 dark:text-zinc-400">
                    {timeAgo(f.foundAt)}
                  </td>
                </tr>
              ))}
              {findings.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-zinc-500">
                    No findings yet.
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
