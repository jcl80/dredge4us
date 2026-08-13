# Builds server/cmd/api from the go.work root. Build context must be
# the repo root (both lib/ and server/ are needed — server depends on
# lib only through the workspace, not a go.mod require).
#
# No build ARG for the binary target — DigitalOcean App Platform's
# declarative builds don't support passing --build-arg through the app
# spec, so cmd/poller gets its own Dockerfile.poller instead of sharing
# this one via an arg. See that file.

FROM golang:1.26-bookworm AS build
WORKDIR /app

COPY go.work go.work.sum ./
COPY lib/go.mod lib/go.sum ./lib/
COPY server/go.mod server/go.sum ./server/
RUN cd lib && go mod download && cd ../server && go mod download

COPY lib ./lib
COPY server ./server

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./server/cmd/api

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
