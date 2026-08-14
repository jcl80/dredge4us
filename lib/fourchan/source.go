package fourchan

import "context"

// Source is anything that can fetch a board's catalog and threads in
// 4chan's own shape. Client (live a.4cdn.org) satisfies it directly;
// lib/foolfuuka.Client (archive mirrors) satisfies it too by having the
// same method signatures, without importing this package back — letting
// the scheduler dispatch each board to whichever source its config
// selects without caring which kind it is.
type Source interface {
	FetchCatalog(ctx context.Context, board, ifModifiedSince string) (Catalog, string, error)
	FetchThread(ctx context.Context, board string, threadNo int, ifModifiedSince string) ([]Post, string, error)
}
