package resolve

import (
	"reflect"
	"testing"
)

// asMap flattens a Result's variables into key->value for easy assertions.
func asMap(r Result) map[string]string {
	m := map[string]string{}
	for _, v := range r.Variables {
		m[v.Key] = v.Value
	}
	return m
}

func TestMerge_BaseOnly(t *testing.T) {
	r := Merge([]ConfigInput{{
		ConfigName: "app", SortOrder: 0,
		Base: []Variable{{Key: "PORT", Value: "3000"}, {Key: "LOG", Value: "info"}},
	}})
	want := map[string]string{"PORT": "3000", "LOG": "info"}
	if got := asMap(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMerge_EnvOverrideWins(t *testing.T) {
	r := Merge([]ConfigInput{{
		ConfigName: "app", SortOrder: 0,
		Base:     []Variable{{Key: "PORT", Value: "3000"}, {Key: "LOG", Value: "info"}},
		Override: []Variable{{Key: "LOG", Value: "debug"}},
	}})
	want := map[string]string{"PORT": "3000", "LOG": "debug"}
	if got := asMap(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	// Provenance: PORT from base, LOG from override.
	for _, v := range r.Variables {
		switch v.Key {
		case "PORT":
			if v.Source != "base" {
				t.Errorf("PORT source = %q, want base", v.Source)
			}
		case "LOG":
			if v.Source != "override" {
				t.Errorf("LOG source = %q, want override", v.Source)
			}
		}
	}
}

func TestMerge_EnvOnlyKey(t *testing.T) {
	r := Merge([]ConfigInput{{
		ConfigName: "app",
		Base:       []Variable{{Key: "PORT", Value: "3000"}},
		Override:   []Variable{{Key: "FEATURE_X", Value: "on"}},
	}})
	want := map[string]string{"PORT": "3000", "FEATURE_X": "on"}
	if got := asMap(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMerge_TombstoneRemovesBaseKey(t *testing.T) {
	r := Merge([]ConfigInput{{
		ConfigName: "app",
		Base:       []Variable{{Key: "DEBUG", Value: "true"}, {Key: "PORT", Value: "3000"}},
		Override:   []Variable{{Key: "DEBUG", Deleted: true}},
	}})
	want := map[string]string{"PORT": "3000"} // DEBUG unset by the env tombstone
	if got := asMap(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMerge_EmptyValueIsPresent(t *testing.T) {
	r := Merge([]ConfigInput{{
		ConfigName: "app",
		Base:       []Variable{{Key: "EMPTY", Value: ""}},
	}})
	if len(r.Variables) != 1 || r.Variables[0].Key != "EMPTY" || r.Variables[0].Value != "" {
		t.Fatalf("empty value should be present: %+v", r.Variables)
	}
}

func TestMerge_CrossConfigCollision_HigherSortOrderWins(t *testing.T) {
	low := ConfigInput{ConfigName: "defaults", SortOrder: 10, Base: []Variable{{Key: "PORT", Value: "3000"}}}
	high := ConfigInput{ConfigName: "overrides", SortOrder: 20, Base: []Variable{{Key: "PORT", Value: "9000"}}}

	// Input order must not matter: higher SortOrder always wins.
	for _, order := range [][]ConfigInput{{low, high}, {high, low}} {
		r := Merge(order)
		if got := asMap(r)["PORT"]; got != "9000" {
			t.Fatalf("PORT = %q, want 9000 (higher sort_order)", got)
		}
		if len(r.Collisions) != 1 {
			t.Fatalf("expected 1 collision, got %d", len(r.Collisions))
		}
		c := r.Collisions[0]
		if c.Key != "PORT" || c.WinningConfig != "overrides" || c.LosingConfig != "defaults" {
			t.Fatalf("unexpected collision: %+v", c)
		}
	}
}

func TestMerge_DeterministicUnderReorder(t *testing.T) {
	a := ConfigInput{ConfigName: "a", SortOrder: 10, Base: []Variable{{Key: "K", Value: "a"}}}
	b := ConfigInput{ConfigName: "b", SortOrder: 20, Base: []Variable{{Key: "K", Value: "b"}}}
	r1 := asMap(Merge([]ConfigInput{a, b}))
	r2 := asMap(Merge([]ConfigInput{b, a}))
	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("merge not deterministic under reorder: %v vs %v", r1, r2)
	}
	if r1["K"] != "b" {
		t.Fatalf("K = %q, want b", r1["K"])
	}
}

func TestMerge_EqualSortOrderTiebreakByName(t *testing.T) {
	// With equal SortOrder, the lexically-greater config name wins (it sorts
	// later in ascending order and overwrites). Deterministic either way.
	a := ConfigInput{ConfigName: "aaa", SortOrder: 5, Base: []Variable{{Key: "K", Value: "from-a"}}}
	z := ConfigInput{ConfigName: "zzz", SortOrder: 5, Base: []Variable{{Key: "K", Value: "from-z"}}}
	r1 := asMap(Merge([]ConfigInput{a, z}))["K"]
	r2 := asMap(Merge([]ConfigInput{z, a}))["K"]
	if r1 != r2 {
		t.Fatalf("equal-sort_order tiebreak not deterministic: %q vs %q", r1, r2)
	}
	if r1 != "from-z" {
		t.Fatalf("tiebreak winner = %q, want from-z", r1)
	}
}

func TestResolveOne_BaseOverrideTombstone(t *testing.T) {
	in := ConfigInput{
		ConfigName: "app",
		SortOrder:  10,
		Base: []Variable{
			{Key: "PORT", Value: "3000"},
			{Key: "LOG", Value: "info"},
			{Key: "DEBUG", Value: "true"},
		},
		Override: []Variable{
			{Key: "LOG", Value: "debug"},  // overrides base
			{Key: "DEBUG", Deleted: true}, // tombstone unsets the inherited base key
			{Key: "EXTRA", Value: "x"},    // env-only key
		},
	}
	r := ResolveOne(in)

	// A single config cannot collide with itself.
	if len(r.Collisions) != 0 {
		t.Fatalf("ResolveOne must not report collisions, got %d", len(r.Collisions))
	}
	want := map[string]string{"PORT": "3000", "LOG": "debug", "EXTRA": "x"} // DEBUG unset
	if got := asMap(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	// Variables come out sorted by key.
	var keys []string
	for _, v := range r.Variables {
		keys = append(keys, v.Key)
	}
	if !reflect.DeepEqual(keys, []string{"EXTRA", "LOG", "PORT"}) {
		t.Fatalf("ResolveOne keys not sorted: %v", keys)
	}
	// Provenance is preserved (same as the whole-project path).
	for _, v := range r.Variables {
		switch v.Key {
		case "PORT":
			if v.Source != "base" {
				t.Errorf("PORT source = %q, want base", v.Source)
			}
		case "LOG":
			if v.Source != "override" {
				t.Errorf("LOG source = %q, want override", v.Source)
			}
		}
	}
}

func TestMerge_NoCrossCollisionWhenSameConfigOverridesBase(t *testing.T) {
	// base+override within the SAME config is not a "collision".
	r := Merge([]ConfigInput{{
		ConfigName: "app",
		Base:       []Variable{{Key: "K", Value: "base"}},
		Override:   []Variable{{Key: "K", Value: "env"}},
	}})
	if len(r.Collisions) != 0 {
		t.Fatalf("intra-config override should not be a collision: %+v", r.Collisions)
	}
	if asMap(r)["K"] != "env" {
		t.Fatalf("K = %q, want env", asMap(r)["K"])
	}
}
