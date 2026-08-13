export type Finding = {
  id: number;
  board: string;
  threadNo: number;
  postNo: number;
  postTime: string;
  detector: string;
  kind: string;
  matchedValue: string;
  note: string | null;
  threadSubject: string;
  threadReplies: number;
  foundAt: string;
};

export function apiBase(): string {
  const base = process.env.API_BASE_URL;
  if (!base) {
    throw new Error("API_BASE_URL is not set");
  }
  return base;
}

// getFindings calls the poller's read-only API server-side. API_BASE_URL
// is not NEXT_PUBLIC_-prefixed on purpose — the browser never talks to
// that API directly. board/kind are omitted to mean "no filter".
export async function getFindings(board?: string, kind?: string): Promise<Finding[]> {
  const params = new URLSearchParams({ limit: "100" });
  if (board) {
    params.set("board", board);
  }
  if (kind) {
    params.set("kind", kind);
  }

  const res = await fetch(`${apiBase()}/findings?${params}`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`findings request failed: ${res.status}`);
  }

  return res.json();
}

// getBoards returns the boards currently watched by the poller.
export async function getBoards(): Promise<string[]> {
  const res = await fetch(`${apiBase()}/boards`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`boards request failed: ${res.status}`);
  }

  return res.json();
}

// getKinds returns the distinct finding kinds present, optionally
// scoped to board — spans every detector (regex kinds like
// "github_url", LLM classes like "ARTIFACT_DROP") and grows as
// detectors are added, so this is a live query, not a hardcoded list.
export async function getKinds(board?: string): Promise<string[]> {
  const params = new URLSearchParams();
  if (board) {
    params.set("board", board);
  }

  const res = await fetch(`${apiBase()}/kinds?${params}`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`kinds request failed: ${res.status}`);
  }

  return res.json();
}

export type GeneralLineage = {
  board: string;
  subjectKey: string;
  threadNo: number;
  threadSubject: string;
  replies: number;
  lastSeenAt: string;
  endedAt: string | null;
  instanceCount: number;
  firstSeenAt: string;
  findingKinds: string[];
};

// getGenerals returns the general-thread lineages tracked for board —
// see lib/general on the Go side for the detection/stitching heuristic.
export async function getGenerals(board: string): Promise<GeneralLineage[]> {
  const res = await fetch(`${apiBase()}/generals?board=${encodeURIComponent(board)}`, {
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`generals request failed: ${res.status}`);
  }

  return res.json();
}
