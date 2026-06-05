// Package resolve implements opentdm's base→environment variable merge — the
// core product nuance. It is pure (no DB, no crypto): it operates on
// already-decrypted variables and is exhaustively golden-tested.
//
// Semantics (see DECISIONS.md):
//   - Two layers: base (env_id IS NULL) → target environment override.
//   - Within a config, an env item overrides the base item for the same key;
//     a tombstone (Deleted) env item removes an inherited base key.
//   - Across configs, the merged maps are flattened into one key/value space;
//     on a key collision the config with the higher SortOrder wins
//     (deterministic, rename-safe), and every collision is reported.
package resolve

import "sort"

// Variable is one decrypted key/value at a single layer of a single config.
type Variable struct {
	Key      string
	Value    string
	IsSecret bool
	Deleted  bool // tombstone: at the env layer, removes an inherited base key
}

// ConfigInput is one variable config's contribution to a resolve: its base
// items and its overrides for the target environment.
type ConfigInput struct {
	ConfigName string
	SortOrder  int
	Base       []Variable // env_id IS NULL
	Override   []Variable // env_id = target environment
}

// Resolved is a single winning key/value in the merged result.
type Resolved struct {
	Key        string
	Value      string
	IsSecret   bool
	ConfigName string // which config supplied the winning value
	Source     string // "base" or "override" within that config
}

// Collision records that the same key was defined by more than one config; the
// higher-SortOrder config won.
type Collision struct {
	Key           string
	WinningConfig string
	LosingConfig  string
}

// Result is the merged outcome: variables sorted by key, plus any collisions.
type Result struct {
	Variables  []Resolved
	Collisions []Collision
}

// Merge resolves the configs for one environment.
func Merge(configs []ConfigInput) Result {
	// Deterministic config order: ascending SortOrder, then ConfigName as a
	// stable tiebreak. Iterating ascending means a later (higher SortOrder)
	// config overwrites an earlier one on collision — i.e. higher wins.
	ordered := make([]ConfigInput, len(configs))
	copy(ordered, configs)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].SortOrder != ordered[j].SortOrder {
			return ordered[i].SortOrder < ordered[j].SortOrder
		}
		return ordered[i].ConfigName < ordered[j].ConfigName
	})

	winners := map[string]Resolved{}
	var collisions []Collision

	for _, cfg := range ordered {
		eff := effective(cfg)
		// Apply this config's effective map over the accumulator.
		for _, key := range sortedKeys(eff) {
			res := eff[key]
			if prev, exists := winners[key]; exists && prev.ConfigName != res.ConfigName {
				collisions = append(collisions, Collision{
					Key:           key,
					WinningConfig: res.ConfigName, // current cfg has >= SortOrder
					LosingConfig:  prev.ConfigName,
				})
			}
			winners[key] = res
		}
	}

	out := Result{Collisions: collisions}
	for _, key := range sortedKeys(winners) {
		out.Variables = append(out.Variables, winners[key])
	}
	// Stable, deterministic collision order.
	sort.SliceStable(out.Collisions, func(i, j int) bool {
		return out.Collisions[i].Key < out.Collisions[j].Key
	})
	return out
}

// effective collapses one config's base + override into a single key→Resolved
// map: base first, then overrides (set or tombstone-delete).
func effective(cfg ConfigInput) map[string]Resolved {
	m := map[string]Resolved{}
	for _, v := range cfg.Base {
		if v.Deleted { // a base tombstone simply means "absent"
			continue
		}
		m[v.Key] = Resolved{Key: v.Key, Value: v.Value, IsSecret: v.IsSecret, ConfigName: cfg.ConfigName, Source: "base"}
	}
	for _, v := range cfg.Override {
		if v.Deleted {
			delete(m, v.Key)
			continue
		}
		m[v.Key] = Resolved{Key: v.Key, Value: v.Value, IsSecret: v.IsSecret, ConfigName: cfg.ConfigName, Source: "override"}
	}
	return m
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
