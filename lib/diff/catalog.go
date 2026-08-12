// Package diff compares two catalog snapshots to find what changed
// between poll cycles: new threads, threads whose activity advanced, and
// threads that dropped out of the catalog entirely.
package diff

import "github.com/jcl80/dredge4us/lib/fourchan"

// Change describes how a board's catalog differs from the previous cycle.
type Change struct {
	New     []fourchan.Thread
	Changed []fourchan.Thread
	Gone    []int
}

// Snapshot flattens a catalog's pages into a lookup keyed by thread
// number — the shape Compute needs to compare cycles.
func Snapshot(catalog fourchan.Catalog) map[int]fourchan.Thread {
	snap := make(map[int]fourchan.Thread)
	for _, page := range catalog {
		for _, t := range page.Threads {
			snap[t.No] = t
		}
	}
	return snap
}

// Compute compares the previous cycle's snapshot against the current one.
// A thread counts as changed if its last_modified advanced or its reply
// count differs — 4chan's API lets either move without the other (e.g. a
// moderator action bumps last_modified with no new reply).
func Compute(prev, curr map[int]fourchan.Thread) Change {
	var c Change

	for no, t := range curr {
		p, existed := prev[no]
		switch {
		case !existed:
			c.New = append(c.New, t)
		case t.LastModified > p.LastModified || t.Replies != p.Replies:
			c.Changed = append(c.Changed, t)
		}
	}

	for no := range prev {
		if _, stillThere := curr[no]; !stillThere {
			c.Gone = append(c.Gone, no)
		}
	}

	return c
}
