package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/codec"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
)

const secretMask = "••••••"

// ListVersions returns version metadata (no payloads) for a (config, layer).
func (s *Service) ListVersions(ctx context.Context, project model.Project, config model.Config, envSlug string) ([]model.ConfigVersion, error) {
	envID, _, err := s.layer(ctx, project.ID, envSlug)
	if err != nil {
		return nil, err
	}
	return s.store.Q().ListVersions(ctx, config.ID, envID)
}

// VersionSnapshot is a decrypted snapshot for one version.
type VersionSnapshot struct {
	Version int
	Kind    string
	Vars    []VarOutput // variable kind
	File    []byte      // file kind
}

// GetVersion returns the decrypted snapshot for a version (version<=0 = current).
// Secret variable values are masked unless reveal is true.
func (s *Service) GetVersion(ctx context.Context, project model.Project, config model.Config, envSlug string, version int, reveal bool) (VersionSnapshot, error) {
	envID, envAAD, err := s.layer(ctx, project.ID, envSlug)
	if err != nil {
		return VersionSnapshot{}, err
	}
	row, err := s.fetchVersionRow(ctx, config.ID, envID, version)
	if err != nil {
		return VersionSnapshot{}, err
	}
	plain, err := s.decryptVersion(project, config, envAAD, row)
	if err != nil {
		return VersionSnapshot{}, err
	}
	out := VersionSnapshot{Version: row.Version, Kind: row.SnapshotKind}
	if row.SnapshotKind == model.KindVariable {
		items, err := codec.ParseVarSnapshot(plain)
		if err != nil {
			return VersionSnapshot{}, err
		}
		for _, it := range items {
			if it.Deleted {
				continue // tombstones aren't shown as effective values
			}
			val := it.Value
			if it.IsSecret && !reveal {
				val = secretMask
			}
			out.Vars = append(out.Vars, VarOutput{Key: it.Key, Value: val, IsSecret: it.IsSecret})
		}
	} else {
		out.File = plain
	}
	return out, nil
}

// fetchVersionRow loads a version row by number (version<=0 = current).
func (s *Service) fetchVersionRow(ctx context.Context, configID uuid.UUID, envID *uuid.UUID, version int) (model.ConfigVersion, error) {
	if version <= 0 {
		return s.store.Q().GetCurrentVersion(ctx, configID, envID)
	}
	return s.store.Q().GetVersion(ctx, configID, envID, version)
}

// decryptVersion decrypts a version snapshot. v1 has a single DEK generation, so
// cipherFor suffices; the row's dek_version is recorded for future rotation.
func (s *Service) decryptVersion(project model.Project, config model.Config, envAAD string, v model.ConfigVersion) ([]byte, error) {
	cipher, err := s.cipherFor(project)
	if err != nil {
		return nil, err
	}
	return cipher.Open(v.SnapshotCiphertext, crypto.VersionAAD(project.ID.String(), envAAD, config.ID.String(), v.SnapshotKind))
}

// VarDiffEntry is one key's change between two variable snapshots.
type VarDiffEntry struct {
	Key       string
	Status    string // added | removed | changed | secret_changed
	From      *string
	To        *string
	WasSecret bool
	IsSecret  bool
}

// DiffResult is a structured diff between two versions of a layer.
type DiffResult struct {
	Kind     string
	From     int
	To       int
	Vars     []VarDiffEntry
	FileDiff string
}

// Diff compares two versions of a (config, layer). from<=0 means the empty
// snapshot (everything added); to<=0 means the current version. Secrets are
// masked; line diffs are refused for secret file configs.
func (s *Service) Diff(ctx context.Context, project model.Project, config model.Config, envSlug string, from, to int) (DiffResult, error) {
	envID, envAAD, err := s.layer(ctx, project.ID, envSlug)
	if err != nil {
		return DiffResult{}, err
	}
	toRow, err := s.fetchVersionRow(ctx, config.ID, envID, to)
	if err != nil {
		return DiffResult{}, err
	}
	toPlain, err := s.decryptVersion(project, config, envAAD, toRow)
	if err != nil {
		return DiffResult{}, err
	}
	var fromPlain []byte
	fromNum := 0
	if from > 0 {
		fromRow, err := s.fetchVersionRow(ctx, config.ID, envID, from)
		if err != nil {
			return DiffResult{}, err
		}
		if fromRow.SnapshotKind != toRow.SnapshotKind {
			return DiffResult{}, invalid("from", "cannot diff across snapshot kinds")
		}
		if fromPlain, err = s.decryptVersion(project, config, envAAD, fromRow); err != nil {
			return DiffResult{}, err
		}
		fromNum = fromRow.Version
	}

	res := DiffResult{Kind: toRow.SnapshotKind, From: fromNum, To: toRow.Version}
	if toRow.SnapshotKind == model.KindVariable {
		res.Vars, err = diffVars(fromPlain, toPlain)
		if err != nil {
			return DiffResult{}, err
		}
	} else if config.IsSecret {
		res.FileDiff = fmt.Sprintf("(secret file — content hidden) %d → %d bytes", len(fromPlain), len(toPlain))
	} else {
		res.FileDiff = unifiedLineDiff(string(fromPlain), string(toPlain))
	}
	return res, nil
}

func diffVars(fromRaw, toRaw []byte) ([]VarDiffEntry, error) {
	fromMap, err := snapshotMap(fromRaw)
	if err != nil {
		return nil, err
	}
	toMap, err := snapshotMap(toRaw)
	if err != nil {
		return nil, err
	}
	keys := map[string]bool{}
	for k := range fromMap {
		keys[k] = true
	}
	for k := range toMap {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var out []VarDiffEntry
	for _, k := range sorted {
		f, fok := fromMap[k]
		t, tok := toMap[k]
		switch {
		case !fok && tok:
			out = append(out, VarDiffEntry{Key: k, Status: "added", To: maskedPtr(t), IsSecret: t.IsSecret})
		case fok && !tok:
			out = append(out, VarDiffEntry{Key: k, Status: "removed", From: maskedPtr(f), WasSecret: f.IsSecret})
		case f.Value != t.Value || f.IsSecret != t.IsSecret:
			status := "changed"
			if f.IsSecret != t.IsSecret {
				status = "secret_changed"
			}
			out = append(out, VarDiffEntry{Key: k, Status: status, From: maskedPtr(f), To: maskedPtr(t), WasSecret: f.IsSecret, IsSecret: t.IsSecret})
		}
	}
	return out, nil
}

func snapshotMap(raw []byte) (map[string]codec.SnapshotItem, error) {
	m := map[string]codec.SnapshotItem{}
	if len(raw) == 0 {
		return m, nil
	}
	items, err := codec.ParseVarSnapshot(raw)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.Deleted {
			continue
		}
		m[it.Key] = it
	}
	return m, nil
}

func maskedPtr(it codec.SnapshotItem) *string {
	v := it.Value
	if it.IsSecret {
		v = secretMask
	}
	return &v
}

// unifiedLineDiff produces a simple line-based diff (LCS), bounded to avoid
// quadratic blowup on huge files.
func unifiedLineDiff(a, b string) string {
	al := strings.Split(a, "\n")
	bl := strings.Split(b, "\n")
	const maxLines = 5000
	if len(al) > maxLines || len(bl) > maxLines {
		return fmt.Sprintf("files differ (%d → %d lines; too large to diff inline)", len(al), len(bl))
	}
	m, n := len(al), len(bl)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if al[i] == bl[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out strings.Builder
	i, j := 0, 0
	for i < m && j < n {
		switch {
		case al[i] == bl[j]:
			out.WriteString("  " + al[i] + "\n")
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			out.WriteString("- " + al[i] + "\n")
			i++
		default:
			out.WriteString("+ " + bl[j] + "\n")
			j++
		}
	}
	for ; i < m; i++ {
		out.WriteString("- " + al[i] + "\n")
	}
	for ; j < n; j++ {
		out.WriteString("+ " + bl[j] + "\n")
	}
	return out.String()
}

// Rollback appends a NEW current version equal to the target's plaintext
// (re-encrypted under the current DEK). Forward-only; never destructive.
func (s *Service) Rollback(ctx context.Context, project model.Project, config model.Config, envSlug string, toVersion int, comment *string, actor *uuid.UUID) (model.ConfigVersion, error) {
	if toVersion <= 0 {
		return model.ConfigVersion{}, invalid("to_version", "must be a positive version number")
	}
	envID, envAAD, err := s.layer(ctx, project.ID, envSlug)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	target, err := s.store.Q().GetVersion(ctx, config.ID, envID, toVersion)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	plain, err := s.decryptVersion(project, config, envAAD, target)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	note := fmt.Sprintf("rollback to v%d", toVersion)
	if comment != nil && *comment != "" {
		note = *comment
	}
	if target.SnapshotKind == model.KindVariable {
		items, err := codec.ParseVarSnapshot(plain)
		if err != nil {
			return model.ConfigVersion{}, err
		}
		inputs := make([]VarInput, 0, len(items))
		for _, it := range items {
			inputs = append(inputs, VarInput{Key: it.Key, Value: it.Value, IsSecret: it.IsSecret, Deleted: it.Deleted})
		}
		return s.SetItems(ctx, project, config, envSlug, inputs, &note, actor)
	}
	return s.SetBlob(ctx, project, config, envSlug, plain, 0, &note, actor)
}
