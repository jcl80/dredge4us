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
