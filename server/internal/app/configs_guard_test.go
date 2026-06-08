package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
)

// Both the env-only creation guard and the ResolveConfig kind guard return before
// touching the store, so a zero-value Service exercises them without a DB.

func TestCreateConfig_EnvOnlyGuardRejectsNonEnv(t *testing.T) {
	if !envOnlyMode {
		t.Skip("env-only mode disabled — non-env creation is allowed")
	}
	rejected := []model.Config{
		{Kind: model.KindVariable, Format: model.FormatProperties, Name: "p"},
		{Kind: model.KindVariable, Format: model.FormatSecret, Name: "s"},
		{Kind: model.KindFile, Format: model.FormatJSON, Name: "j"},
		{Kind: model.KindFile, Format: model.FormatCSV, Name: "c"},
		{Kind: model.KindFile, Format: model.FormatXML, Name: "x"},
	}
	for _, c := range rejected {
		_, err := (&Service{}).CreateConfig(context.Background(), model.User{}, uuid.New(), c)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("CreateConfig(%s/%s) = %v, want *ValidationError", c.Kind, c.Format, err)
		}
	}
}

func TestResolveConfig_RejectsFileConfig(t *testing.T) {
	_, err := (&Service{}).ResolveConfig(
		context.Background(),
		model.Project{},
		model.Config{Kind: model.KindFile, Format: model.FormatJSON, Name: "seed"},
		uuid.New(),
	)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("ResolveConfig(file) = %v, want *ValidationError", err)
	}
}
