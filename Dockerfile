# Builds either server binary from the go.work root. Build context must
# be the repo root (both lib/ and server/ are needed — server depends on
# lib only through the workspace, not a go.mod require).
#
#   docker build --build-arg CMD=api    -t dredge4us-api    .
#   docker build --build-arg CMD=poller -t dredge4us-poller .

FROM golang:1.26-bookworm AS build
WORKDIR /app

COPY go.work go.work.sum ./
COPY lib/go.mod lib/go.sum ./lib/
COPY server/go.mod server/go.sum ./server/
RUN cd lib && go mod download && cd ../server && go mod download

COPY lib ./lib
COPY server ./server

ARG CMD=api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./server/cmd/${CMD}

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
