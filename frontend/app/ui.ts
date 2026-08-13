export function timeAgo(iso: string): string {
  const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function pillClass(active: boolean): string {
  return `rounded-full px-3 py-1 ${
    active
      ? "bg-black text-white dark:bg-white dark:text-black"
      : "bg-black/[.05] dark:bg-white/[.08]"
  }`;
}

// Color grouping for LLM-classified kinds, by rough severity/category.
// Regex kinds (lowercase, e.g. "github_url") aren't judgment calls, so
// they stay neutral gray via the fallback.
const KIND_COLORS: Record<string, string> = {
  ARTIFACT_DROP: "bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300",
  TOOLING: "bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300",
  CAPABILITY_CLAIM: "bg-purple-100 text-purple-800 dark:bg-purple-900/40 dark:text-purple-300",
  LEAK_DOC: "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300",
  INSIDER_TIP: "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300",
  MISUSE_DEMAND: "bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300",
  ACCOUNT_COMPROMISE: "bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300",
  VULN_DISCLOSURE: "bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300",
  CORP_NEWS: "bg-teal-100 text-teal-800 dark:bg-teal-900/40 dark:text-teal-300",
  SENTIMENT_SHIFT: "bg-teal-100 text-teal-800 dark:bg-teal-900/40 dark:text-teal-300",
  RECRUITMENT_CALL: "bg-pink-100 text-pink-800 dark:bg-pink-900/40 dark:text-pink-300",
};

export function kindBadgeClass(kind: string): string {
  return (
    KIND_COLORS[kind] ??
    "bg-zinc-100 text-zinc-700 dark:bg-white/[.08] dark:text-zinc-300"
  );
}
