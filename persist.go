package ditto

import (
	"encoding/gob"
	"fmt"
	"io"
)

// snapshot is the persisted form of an Index: its k and the (id, fingerprint)
// pairs. The banded tables are not stored — [LoadIndex] rebuilds them — so the
// file stays small (8 bytes + the id per entry). gob is used rather than JSON
// because a 64-bit fingerprint above 2^53 would lose precision as a JSON number.
type snapshot struct {
	K   int
	IDs []string
	FPs []Fingerprint
}

// Save writes the index to w with encoding/gob, so a large fleet index can be
// persisted between runs instead of rebuilt from every fingerprint each time.
func (ix *Index) Save(w io.Writer) error {
	return gob.NewEncoder(w).Encode(snapshot{K: ix.k, IDs: ix.ids, FPs: ix.fps})
}

// LoadIndex reads an index written by [Index.Save], rebuilding the banded tables.
// The reconstructed index is byte-for-byte equivalent to the original (same k,
// same entries, same Near/Clusters results).
// maxSnapshotBytes bounds how much LoadIndex reads from r, so a hostile or corrupt
// gob stream cannot drive unbounded allocation. 1 GiB is far beyond any realistic
// in-memory index while still capping the damage; LoadIndex is meant for trusted
// snapshots regardless.
const maxSnapshotBytes = 1 << 30

func LoadIndex(r io.Reader) (*Index, error) {
	var s snapshot
	if err := gob.NewDecoder(io.LimitReader(r, maxSnapshotBytes)).Decode(&s); err != nil {
		return nil, err
	}
	if len(s.IDs) != len(s.FPs) {
		return nil, fmt.Errorf("ditto: corrupt index snapshot: %d ids, %d fingerprints", len(s.IDs), len(s.FPs))
	}
	ix := NewIndex(s.K)
	for i := range s.IDs {
		ix.Add(s.IDs[i], s.FPs[i])
	}
	return ix, nil
}
