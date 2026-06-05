package httpapi

import (
	"time"

	"github.com/opentdm/opentdm/server/internal/model"
)

type projectDTO struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func toProjectDTO(p model.Project) projectDTO {
	return projectDTO{ID: p.ID.String(), Slug: p.Slug, Name: p.Name, Description: p.Description, CreatedAt: p.CreatedAt}
}

type environmentDTO struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Rank      int    `json:"rank"`
	IsDefault bool   `json:"is_default"`
}

func toEnvironmentDTO(e model.Environment) environmentDTO {
	return environmentDTO{ID: e.ID.String(), Slug: e.Slug, Name: e.Name, Rank: e.Rank, IsDefault: e.IsDefault}
}

type configDTO struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Format      string    `json:"format"`
	Name        string    `json:"name"`
	SortOrder   int       `json:"sort_order"`
	Description string    `json:"description"`
	IsSecret    bool      `json:"is_secret"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toConfigDTO(c model.Config) configDTO {
	tags := c.Tags
	if tags == nil {
		tags = []string{}
	}
	return configDTO{
		ID: c.ID.String(), Kind: c.Kind, Format: c.Format, Name: c.Name, SortOrder: c.SortOrder,
		Description: c.Description, IsSecret: c.IsSecret, Tags: tags, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

type tokenDTO struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scope      string     `json:"scope"`
	EnvIDs     []string   `json:"environment_ids"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func toTokenDTO(t model.Token) tokenDTO {
	ids := make([]string, 0, len(t.EnvIDs))
	for _, id := range t.EnvIDs {
		ids = append(ids, id.String())
	}
	return tokenDTO{
		ID: t.ID.String(), Name: t.Name, Prefix: t.Prefix, Scope: t.Scope, EnvIDs: ids,
		ExpiresAt: t.ExpiresAt, LastUsedAt: t.LastUsedAt, RevokedAt: t.RevokedAt, CreatedAt: t.CreatedAt,
	}
}
