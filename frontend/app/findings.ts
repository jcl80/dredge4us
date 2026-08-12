export type Finding = {
  id: number;
  board: string;
  threadNo: number;
  postNo: number;
  postTime: string;
  detector: string;
  kind: string;
  matchedValue: string;
  threadSubject: string;
  threadReplies: number;
  foundAt: string;
};

// getFindings calls the poller's read-only API server-side. API_BASE_URL
// is not NEXT_PUBLIC_-prefixed on purpose — the browser never talks to
// that API directly.
export async function getFindings(): Promise<Finding[]> {
  const base = process.env.API_BASE_URL;
  if (!base) {
    throw new Error("API_BASE_URL is not set");
  }

  const res = await fetch(`${base}/findings?limit=100`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`findings request failed: ${res.status}`);
  }

  return res.json();
}
