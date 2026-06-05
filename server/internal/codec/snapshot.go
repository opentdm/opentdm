package codec

import (
	"encoding/json"
	"sort"
)

// SnapshotItem is one variable in a version snapshot.
type SnapshotItem struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
	Deleted  bool   `json:"deleted"`
}

type varSnapshot struct {
	V     int            `json:"v"`
	Items []SnapshotItem `json:"items"`
}

// CanonicalVarSnapshot serializes a variable layer deterministically (items
// sorted by key, fixed field order, deleted values normalized to empty) so the
// resulting bytes are a stable equality signal for content hashing and diff.
func CanonicalVarSnapshot(items []SnapshotItem) ([]byte, error) {
	out := make([]SnapshotItem, len(items))
	copy(out, items)
	for i := range out {
		if out[i].Deleted {
			out[i].Value = ""
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return json.Marshal(varSnapshot{V: 1, Items: out})
}

// ParseVarSnapshot decodes a canonical variable snapshot back into items.
func ParseVarSnapshot(raw []byte) ([]SnapshotItem, error) {
	var s varSnapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return s.Items, nil
}
