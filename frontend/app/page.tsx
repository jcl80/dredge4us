import Link from "next/link";
import { getBoards, getFindings } from "./findings";
import { pillClass, timeAgo } from "./ui";

function isLinkable(value: string): boolean {
  return value.startsWith("http://") || value.startsWith("https://") || value.startsWith("magnet:");
}

function boardHref(board?: string): string {
  return board ? `/?board=${board}` : "/";
}

export default async function Home(props: PageProps<"/">) {
  const searchParams = await props.searchParams;
  const board = typeof searchParams.board === "string" ? searchParams.board : undefined;

  const [boards, findings] = await Promise.all([getBoards(), getFindings(board)]);

  return (
    <div className="flex-1 bg-zinc-50 dark:bg-black">
      <main className="mx-auto max-w-5xl px-6 py-10">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-semibold text-black dark:text-zinc-50">
            Findings
          </h1>
          <Link href="/generals" className="text-sm text-zinc-600 hover:underline dark:text-zinc-400">
            Generals &rarr;
          </Link>
        </div>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          {findings.length} most recent artifact detections.
        </p>

        <nav className="mt-4 flex flex-wrap gap-2 text-sm">
          <Link href={boardHref()} className={pillClass(!board)}>
            All
          </Link>
          {boards.map((b) => (
            <Link key={b} href={boardHref(b)} className={pillClass(board === b)}>
              /{b}/
            </Link>
          ))}
        </nav>

        <div className="mt-6 overflow-x-auto rounded-lg border border-black/[.08] dark:border-white/[.145]">
          <table className="w-full text-left text-sm">
            <thead className="bg-black/[.03] dark:bg-white/[.04]">
              <tr>
                <th className="px-4 py-2 font-medium">Board</th>
                <th className="px-4 py-2 font-medium">Kind</th>
                <th className="px-4 py-2 font-medium">Matched value</th>
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
                  <td className="px-4 py-2 whitespace-nowrap">{f.kind}</td>
                  <td className="max-w-md truncate px-4 py-2 font-mono text-xs">
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
                  </td>
                  <td className="max-w-xs truncate px-4 py-2">
                    {f.threadSubject || `#${f.threadNo}`}
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
