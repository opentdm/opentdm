package resolve

import "testing"

// TestMerge_EmptyOverrideShadowsBase pins a subtle merge semantic: a present,
// non-deleted env override with an empty value OVERRIDES (hides) the inherited
// base value — it is a placeholder, not "inherit from base". (Only a tombstone
// restores base-absence.)
func TestMerge_EmptyOverrideShadowsBase(t *testing.T) {
	cfg := ConfigInput{
		ConfigName: "app",
		SortOrder:  10,
		Base:       []Variable{{Key: "DB_HOST", Value: "localhost"}},
		Override:   []Variable{{Key: "DB_HOST", Value: ""}}, // empty, not deleted
	}
	res := Merge([]ConfigInput{cfg})
	if len(res.Variables) != 1 {
		t.Fatalf("expected DB_HOST to be present, got %d vars", len(res.Variables))
	}
	got := res.Variables[0]
	if got.Value != "" || got.Source != "override" {
		t.Errorf("empty override should shadow base with value=\"\" from override, got value=%q source=%q", got.Value, got.Source)
	}

	// Contrast: a tombstone override restores base-absence (key disappears).
	cfg.Override = []Variable{{Key: "DB_HOST", Deleted: true}}
	if vars := Merge([]ConfigInput{cfg}).Variables; len(vars) != 0 {
		t.Errorf("tombstone override should remove the key, got %+v", vars)
	}
}
