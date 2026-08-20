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
// Desaturated so a list of twelve badges doesn't read as a carnival.
// Regex kinds (lowercase, e.g. "github_url") aren't judgment calls, so
// they stay neutral gray via the fallback.
const KIND_COLORS: Record<string, string> = {
  ARTIFACT_DROP: "bg-[#e7edf7] text-[#2b4c86]",
  TOOLING: "bg-[#e7edf7] text-[#2b4c86]",
  CAPABILITY_CLAIM: "bg-[#efe9f6] text-[#5b3f83]",
  LEAK_DOC: "bg-[#f7eede] text-[#8a5a11]",
  INSIDER_TIP: "bg-[#f7eede] text-[#8a5a11]",
  MISUSE_DEMAND: "bg-[#f8e6e2] text-[#9c3520]",
  ACCOUNT_COMPROMISE: "bg-[#f8e6e2] text-[#9c3520]",
  VULN_DISCLOSURE: "bg-[#f8e6e2] text-[#9c3520]",
  CORP_NEWS: "bg-[#e3f0ec] text-[#1c6650]",
  SENTIMENT_SHIFT: "bg-[#e3f0ec] text-[#1c6650]",
  RECRUITMENT_CALL: "bg-[#f7e8ef] text-[#8a3358]",
};

export function kindBadgeClass(kind: string): string {
  return KIND_COLORS[kind] ?? "bg-[#f0eeeb] text-[#5c554d]";
}
