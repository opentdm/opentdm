package cli

import (
	"reflect"
	"sort"
	"testing"
)

func TestSplitDashDash(t *testing.T) {
	before, after := splitDashDash([]string{"--env", "staging", "--", "npm", "test"})
	if !reflect.DeepEqual(before, []string{"--env", "staging"}) {
		t.Errorf("before = %v", before)
	}
	if !reflect.DeepEqual(after, []string{"npm", "test"}) {
		t.Errorf("after = %v", after)
	}
	b2, a2 := splitDashDash([]string{"--env", "staging"})
	if a2 != nil || !reflect.DeepEqual(b2, []string{"--env", "staging"}) {
		t.Errorf("no dashdash: before=%v after=%v", b2, a2)
	}
}

func TestMergeEnv_ResolvedWins(t *testing.T) {
	base := []string{"PATH=/usr/bin", "LOG=info", "HOME=/root"}
	vars := map[string]string{"LOG": "debug", "NEW": "1"}
	got := mergeEnv(base, vars)
	sort.Strings(got)
	want := []string{"HOME=/root", "LOG=debug", "NEW=1", "PATH=/usr/bin"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeEnv = %v, want %v", got, want)
	}
}
