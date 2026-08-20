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

// kindBadgeChipClass is kindBadgeClass plus the fixed badge sizing/type
// shared by every place a kind badge renders (findings list rows, the
// detail header, sibling findings, ...).
export function kindBadgeChipClass(kind: string): string {
  return `rounded-[3px] px-1.5 py-0.5 font-mono text-[10px] tracking-[0.03em] ${kindBadgeClass(kind)}`;
}

// confidenceInk is the severity-ink color for a finding's confidence bar
// — only used there, never as a general severity indicator elsewhere.
export function confidenceInk(confidence: number): string {
  if (confidence >= 0.75) return "#9c3520";
  if (confidence >= 0.6) return "#a8620f";
  return "#8a827a";
}

export function confidencePct(confidence: number): string {
  return `${Math.round(confidence * 100)}%`;
}

// detectorLabel maps a Finding.detector value to the display name the
// design uses ("regex", "llm") — the stored value is the detector's own
// Name(), which doesn't match that vocabulary.
const DETECTOR_LABELS: Record<string, string> = {
  artifact: "regex",
  llm_classify: "llm",
};

export function detectorLabel(detector: string): string {
  return DETECTOR_LABELS[detector] ?? detector;
}

// kindAccentColor is each kind's badge text color as a raw hex, for
// contexts that can't use a Tailwind class (e.g. an inline style on a
// bar fill) — KIND_COLORS above has to stay literal strings for
// Tailwind's static scanner to see them, so this is a second table,
// kept in sync with it by hand.
const KIND_ACCENT: Record<string, string> = {
  ARTIFACT_DROP: "#2b4c86",
  TOOLING: "#2b4c86",
  CAPABILITY_CLAIM: "#5b3f83",
  LEAK_DOC: "#8a5a11",
  INSIDER_TIP: "#8a5a11",
  MISUSE_DEMAND: "#9c3520",
  ACCOUNT_COMPROMISE: "#9c3520",
  VULN_DISCLOSURE: "#9c3520",
  CORP_NEWS: "#1c6650",
  SENTIMENT_SHIFT: "#1c6650",
  RECRUITMENT_CALL: "#8a3358",
};

export function kindAccentColor(kind: string): string {
  return KIND_ACCENT[kind] ?? "#5c554d";
}
