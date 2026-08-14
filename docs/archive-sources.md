# Archive sources — design

## Goal

Increase the throughput of what can be analyzed by pulling boards from
third-party 4chan archives instead of live `a.4cdn.org`, where those
archives are close to real-time. This does not raise 4chan's own 1 req/s
ceiling — it sidesteps it by spreading watched boards across several
*independent* services, each with its own host and its own rate limit.
Text only: no image/attachment fields, matching `lib/fourchan`'s existing
scope.

## Board → archive mapping

All four candidates run FoolFuuka, the same archive software, which is why
they can share one client implementation. Reachability was checked
directly (see **Cloudflare findings** below) with a descriptive User-Agent
identifying this project and a contact address — standard etiquette for
an automated client, and the fix for what first looked like a
Cloudflare block:

| Archive           | Host                  | Boards           | Reachable? |
|--------------------|------------------------|-------------------|------------|
| desuarchive         | desuarchive.org        | /his/, /k/, /g/, /int/ | Yes — 200s from `/_/api/chan/*` with a descriptive UA |
| palanq              | archive.palanq.win     | /news/            | Yes — same |
| 4plebs             | archive.4plebs.org    | /pol/, /tv/, /o/  | **No** — Cloudflare managed challenge (`cf-mitigated: challenge`), UA-independent |
| Archived.Moe        | archived.moe            | /biz/, /sci/, /mu/ | **No** — same |

Only desuarchive and palanq are wired up in the initial implementation.
/pol/, /tv/, /o/, /biz/, /sci/, /mu/ keep polling live 4chan for now —
that's the existing fallback behavior for any board with no working
archive mapping, not a special case. Revisit 4plebs/Archived.Moe if their
operators offer an API-key path for automated consumers (see **What this
design deliberately does not do**); routing around a managed challenge
itself is not something this design attempts.

Board coverage above is what the user proposed as a starting mapping; it
still needs confirming against each archive's own board list before
being treated as durable — FoolFuuka archives periodically add/drop
boards as volunteer agreements with board communities change.

Boards with no archive mapping keep polling live 4chan via `a.4cdn.org`,
exactly as today.

## Architecture

Mirrors the existing `lib/fourchan` shape:

- A new `lib` package (e.g. `lib/foolfuuka`) with a `Client` and a
  `Limiter`, structured like `fourchan.Client`/`fourchan.Limiter`.
- **One `Limiter` per archive host**, not one shared globally across
  hosts. This is the actual throughput win: desuarchive and palanq each
  enforce their own per-IP limit independently, so running one limiter
  per host means the aggregate request rate is the *sum* of three
  independent ceilings (4chan live + 2 archives) instead of being
  bottlenecked by one shared limiter. It is still one request at a time
  per host, same as `fourchan.Limiter` does for `a.4cdn.org` today — each
  site's own limit is respected, not raised. Neither archive publishes an
  explicit rate (no `X-RateLimit-*` response headers observed), so start
  conservative — reuse `fourchan.RequestInterval` (1 req/s) per host —
  and have the `Limiter` back off on 429/503 rather than assuming a
  faster rate is safe.
- Send a descriptive `User-Agent` identifying the project and a contact
  address on every request (e.g. `dredge4us/0.1
  (+mailto:jcambrac@gmail.com; 4chan findings monitor)`). This isn't
  optional politeness — it's the difference between desuarchive/palanq
  serving normal 200s and looking like an anonymous scraper.
- `Store.LastModified`/`SetLastModified` are already keyed by arbitrary
  URL (see `lib/store/store.go`), so conditional GETs (If-Modified-Since)
  work unchanged for archive URLs — no store changes needed.
- FoolFuuka's JSON API is structurally close to 4chan's own (catalog-like
  index endpoint, thread endpoint keyed by board+thread id), so
  `FetchCatalog`/`FetchThread` on the new client can mirror
  `fourchan.Client`'s method shapes, easing the scheduler-side diff.

## Config

Static config, extending the existing `POLLER_BOARDS` format
(`server/internal/config`). Each entry gains an optional source segment;
omitted means live 4chan (today's default and behavior):

```
POLLER_BOARDS=his:desuarchive:20s,k:desuarchive:20s,g:desuarchive:20s,int:desuarchive:20s,news:palanq:20s,pol:20s,tv:20s,o:20s,biz:20s,sci:20s,mu:20s
```

Entries with no source segment (`pol:20s`, `tv:20s`, ...) poll live, as
they do today — that covers /pol/, /tv/, /o/, /biz/, /sci/, /mu/ until
4plebs/Archived.Moe access is resolved. `config.Board` gains a `Source`
field; `parseBoards` gains a small archive-name → host lookup table
(`desuarchive`, `palanq` for now).

## Mode: replace per board

A board with an archive mapping polls *only* that archive — it does not
also poll live 4chan. This keeps the scheduler's one-goroutine-per-board
model unchanged (`scheduler.watchBoard`): the goroutine just gets handed
whichever client (`fourchan.Client` or the new archive client) matches
its `config.Board.Source` at construction time, with no dedup logic
needed between two sources for the same board.

## What this design deliberately does not do

Distributing boards across independently-operated archives is spreading
load across genuinely separate services, each honoring its own published
per-IP limit — not evasion of any single limit. What's explicitly out of
scope, and won't be built here: rotating IPs/proxies to pull more than
one IP's worth of allowance from a *single* archive, or anything that
solves an anti-bot challenge to impersonate a browser. If more headroom
on one archive is needed later, the legitimate path is asking that
archive's operator for elevated/API-key access, not working around their
per-IP throttle.

## Cloudflare findings

Initial requests with WebFetch and a bare `curl` UA (`curl/8.x`) got
Cloudflare "Please wait" challenge pages (403) from all four archives.
Retesting with a descriptive UA and contact info resolved it for two of
four:

- **desuarchive.org, archive.palanq.win**: 200s with real JSON from
  `/_/api/chan/index/?board=X&page=N` (catalog-equivalent) and
  `/_/api/chan/thread/?board=X&num=N` once the UA was set. No
  `cf-mitigated` header — Cloudflare wasn't challenging these at all;
  it was serving a UA-based bot page to unidentified clients, and a
  proper identifying UA is exactly what a well-behaved automated client
  should send regardless.
- **archive.4plebs.org, archived.moe**: still 403 on every path,
  UA change or not, with `cf-mitigated: challenge` in the response and a
  CSP referencing `challenges.cloudflare.com` (Turnstile). This is an
  active managed challenge, not a UA sniff — passing it needs solving a
  JS challenge, which is the anti-bot-evasion line this design won't
  cross. These two are left out of the initial mapping; see **Board →
  archive mapping** above.

Caveat: all of the above was tested from a phone hotspot's residential
mobile IP, not the eventual production egress. Cloudflare's challenge
decisions lean heavily on IP reputation, and datacenter/cloud IP ranges
(where the real poller will run) get challenged far more readily than
residential ones — so these results, especially the two "reachable"
verdicts, need re-testing from the actual production server before being
trusted. It's plausible desuarchive/palanq challenge a datacenter IP even
with a good UA, or that 4plebs/Archived.Moe don't challenge one at all.
Also untested: whether request volume over time changes what
desuarchive/palanq allow (only a handful of one-off requests were made
while drafting this).

The client scaffold that follows this doc targets desuarchive and
palanq's confirmed-working `/_/api/chan/index/` and `/_/api/chan/thread/`
endpoints.
