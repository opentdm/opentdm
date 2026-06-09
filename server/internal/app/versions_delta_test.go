package app

import (
	"testing"

	"github.com/opentdm/opentdm/server/internal/codec"
)

func TestDeltaCounts(t *testing.T) {
	item := func(v string, secret bool) codec.SnapshotItem {
		return codec.SnapshotItem{Value: v, IsSecret: secret}
	}
	tests := []struct {
		name                    string
		from, to                map[string]codec.SnapshotItem
		added, changed, removed int
	}{
		{
			name:  "nil from counts all as added",
			from:  nil,
			to:    map[string]codec.SnapshotItem{"A": item("1", false), "B": item("2", false)},
			added: 2,
		},
		{
			name:    "value change is a modification",
			from:    map[string]codec.SnapshotItem{"A": item("1", false)},
			to:      map[string]codec.SnapshotItem{"A": item("2", false)},
			changed: 1,
		},
		{
			name:    "secret-flag flip counts as changed even with same value",
			from:    map[string]codec.SnapshotItem{"A": item("1", false)},
			to:      map[string]codec.SnapshotItem{"A": item("1", true)},
			changed: 1,
		},
		{
			name:    "removed key",
			from:    map[string]codec.SnapshotItem{"A": item("1", false), "B": item("2", false)},
			to:      map[string]codec.SnapshotItem{"A": item("1", false)},
			removed: 1,
		},
		{
			name:    "mixed add/change/remove",
			from:    map[string]codec.SnapshotItem{"A": item("1", false), "B": item("2", false)},
			to:      map[string]codec.SnapshotItem{"A": item("9", false), "C": item("3", false)},
			added:   1,
			changed: 1,
			removed: 1,
		},
		{
			name: "no change",
			from: map[string]codec.SnapshotItem{"A": item("1", false)},
			to:   map[string]codec.SnapshotItem{"A": item("1", false)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deltaCounts(tt.from, tt.to)
			if got.Added != tt.added || got.Changed != tt.changed || got.Removed != tt.removed {
				t.Errorf("deltaCounts = {Added:%d Changed:%d Removed:%d}, want {Added:%d Changed:%d Removed:%d}",
					got.Added, got.Changed, got.Removed, tt.added, tt.changed, tt.removed)
			}
		})
	}
}
