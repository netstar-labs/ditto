package ditto

import (
	"cmp"
	"slices"
)

// Match is one near-duplicate hit from [Index.Near].
type Match struct {
	ID          string
	Fingerprint Fingerprint
	Distance    int
}

// Index finds near-duplicates in better-than-O(n²) time with banded lookup: a
// block-partition bucketing scheme, per the pigeonhole guarantee used in the
// Google near-dup crawl paper (no bit permutation is performed). Built for a
// maximum Hamming distance k, it splits each 64-bit fingerprint into k+1 blocks;
// by the pigeonhole principle any two fingerprints within k bits share at least
// one identical block, so only fingerprints colliding on a block are compared.
//
// The sub-quadratic guarantee assumes band buckets stay small, which holds when
// near-duplicates are the exception. A corpus dominated by many mutually
// near-duplicate documents (e.g. a mass-identical spam blast) makes every member
// share a bucket with every other, so Near/Clusters degrade toward O(n²) for that
// cluster — inherent to banding, not a bug; see BenchmarkClustersAllDuplicate in
// bench_test.go.
//
// Add fingerprints, then query with Near or group the whole set with Clusters. An
// Index is not safe for concurrent Add; build it, then read. Violating that
// contract is undefined behavior, but note its character changed: candidates
// (used by Near) once deduped through a map, whose out-of-range key access
// never panics, so a concurrent Add raced with a read surfaced only as Go's
// own unrecoverable "concurrent map read and map write" fatal error — never a
// recoverable panic. It now dedupes through a slice sized from a snapshot of
// Len, so the same misuse can also surface as an ordinary (recoverable) index-
// out-of-range panic. Both signal the same contract violation; a caller using
// recover() to contain misuse should not treat either as a softer signal than
// the other.
//
// The zero value is not usable: always construct via [NewIndex] or [LoadIndex].
// [Index.Add] panics on a zero-value Index rather than silently accepting
// entries that would never reach a band table.
type Index struct {
	k      int
	bounds []uint             // block bit boundaries, len k+2
	tables []map[uint64][]int // one per block: block-bits -> entry indices
	ids    []string
	fps    []Fingerprint
}

// NewIndex creates an index whose Near/Clusters resolve near-duplicates up to a
// Hamming distance of k (k >= 0).
func NewIndex(k int) *Index {
	// each of the k+1 blocks needs at least 1 bit, so 63 is the max
	k = min(max(k, 0), 63)
	blocks := k + 1
	bounds := make([]uint, blocks+1)
	for i := 0; i <= blocks; i++ {
		bounds[i] = uint(i * 64 / blocks)
	}
	tables := make([]map[uint64][]int, blocks)
	for i := range tables {
		tables[i] = make(map[uint64][]int)
	}
	return &Index{k: k, bounds: bounds, tables: tables}
}

// K is the maximum Hamming distance the index was built to resolve.
func (ix *Index) K() int { return ix.k }

// Len is the number of fingerprints indexed.
func (ix *Index) Len() int { return len(ix.fps) }

func (ix *Index) block(fp Fingerprint, i int) uint64 {
	lo, hi := ix.bounds[i], ix.bounds[i+1]
	return (uint64(fp) >> lo) & ((uint64(1) << (hi - lo)) - 1)
}

// Add inserts a fingerprint under an id. Ids need not be unique, but duplicates
// are reported separately. Add panics if ix is a zero-value Index (not built
// via [NewIndex] or [LoadIndex]) — see the Index doc comment.
func (ix *Index) Add(id string, fp Fingerprint) {
	if ix.tables == nil {
		panic("ditto: Index.Add called on a zero-value Index; construct with NewIndex or LoadIndex")
	}
	idx := len(ix.ids)
	ix.ids = append(ix.ids, id)
	ix.fps = append(ix.fps, fp)
	for i := range ix.tables {
		key := ix.block(fp, i)
		ix.tables[i][key] = append(ix.tables[i][key], idx)
	}
}

// candidates returns the deduped entry indices sharing at least one block with fp.
func (ix *Index) candidates(fp Fingerprint) []int {
	visited := make([]bool, len(ix.fps)) // call-local: as safe for concurrent Near as the map it replaces
	var out []int
	for i := range ix.tables {
		for _, idx := range ix.tables[i][ix.block(fp, i)] {
			if !visited[idx] {
				visited[idx] = true
				out = append(out, idx)
			}
		}
	}
	return out
}

// clampK caps a queried k to the index's build-time K.
func (ix *Index) clampK(k int) int {
	if k > ix.k {
		return ix.k
	}
	return k
}

// Near returns every indexed entry within k bits of fp, nearest first. k is
// clamped to the index's build-time K.
func (ix *Index) Near(fp Fingerprint, k int) []Match {
	k = ix.clampK(k)
	var out []Match
	for _, idx := range ix.candidates(fp) {
		if d := Distance(fp, ix.fps[idx]); d <= k {
			out = append(out, Match{ID: ix.ids[idx], Fingerprint: ix.fps[idx], Distance: d})
		}
	}
	slices.SortFunc(out, func(a, b Match) int {
		return cmp.Or(cmp.Compare(a.Distance, b.Distance), cmp.Compare(a.ID, b.ID))
	})
	return out
}

// Clusters groups the indexed fingerprints into near-duplicate families by
// union-find over pairs within k bits, returning only clusters of at least min
// members (min <= 1 includes singletons). Output is deterministic: ids sorted
// within each cluster, clusters by descending size then first id. k is clamped to
// the index's build-time K.
func (ix *Index) Clusters(k, min int) [][]string {
	k = ix.clampK(k)
	n := len(ix.ids)
	u := newUnionFind(n)
	// One reusable generation-stamped visited slice for the whole call: visited[j]
	// == gen means j was already considered for the current source, replacing the
	// per-source candidates() map (Near keeps its own map, so it stays safe for
	// concurrent use).
	visited := make([]int, n)
	for idx := range ix.fps {
		gen := idx + 1
		fp := ix.fps[idx]
		for b := range ix.tables {
			for _, j := range ix.tables[b][ix.block(fp, b)] {
				if visited[j] == gen {
					continue
				}
				visited[j] = gen
				if j != idx && Distance(fp, ix.fps[j]) <= k {
					u.union(idx, j)
				}
			}
		}
	}
	groups := make(map[int][]string)
	for idx, id := range ix.ids {
		root := u.find(idx)
		groups[root] = append(groups[root], id)
	}
	out := make([][]string, 0, len(groups))
	for _, g := range groups {
		if len(g) >= min {
			slices.Sort(g)
			out = append(out, g)
		}
	}
	slices.SortFunc(out, func(a, b []string) int {
		// Total order: descending size, then full lexicographic content. Comparing
		// only a[0] tied when two clusters shared a smallest id (Add permits
		// duplicate ids), leaving their order at the mercy of map iteration.
		return cmp.Or(cmp.Compare(len(b), len(a)), slices.Compare(a, b))
	})
	return out
}

// unionFind is a disjoint-set forest with path compression and union by rank.
type unionFind struct{ parent, rank []int }

func newUnionFind(n int) *unionFind {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &unionFind{parent: p, rank: make([]int, n)}
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra == rb {
		return
	}
	if u.rank[ra] < u.rank[rb] {
		ra, rb = rb, ra
	}
	u.parent[rb] = ra
	if u.rank[ra] == u.rank[rb] {
		u.rank[ra]++
	}
}
