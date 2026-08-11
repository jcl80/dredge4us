# Build constraints

## Two modules, deliberately. Never one at the root.

`lib/` and `server/` are separate Go modules. Do NOT run `go mod init` at the
repo root — a root module would put `lib` and `server` in one dependency graph
and silently destroy the boundary below.

    lib/go.mod      module github.com/jcl80/dredge4us/lib
    server/go.mod   module github.com/jcl80/dredge4us/server
    go.work         go work init ./lib ./server   (committed)

Why: `lib` is a reusable component that outside consumers import with
`go get github.com/jcl80/dredge4us/lib`. With one root module they would
inherit the server's entire dependency list — Postgres driver and all — for a
package that parses HTML.