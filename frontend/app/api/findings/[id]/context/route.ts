import { NextResponse } from "next/server";
import { apiBase } from "../../../../findings";

// Proxies the poller API's /findings/{id}/context so the browser never
// talks to it directly — API_BASE_URL stays server-side, same rule as
// every server-fetched page in this app.
export async function GET(_req: Request, ctx: RouteContext<"/api/findings/[id]/context">) {
  const { id } = await ctx.params;
  const res = await fetch(`${apiBase()}/findings/${id}/context`, { cache: "no-store" });
  if (!res.ok) {
    return NextResponse.json({ error: "finding context request failed" }, { status: res.status });
  }
  return NextResponse.json(await res.json());
}
