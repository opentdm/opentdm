package httpapi

import (
	"time"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/model"
)

type projectDTO struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	YourRole    string    `json:"your_role,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func toProjectDTO(p model.Project) projectDTO {
	return projectDTO{ID: p.ID.String(), Slug: p.Slug, Name: p.Name, Description: p.Description, CreatedAt: p.CreatedAt}
}

func toProjectDTOWithRole(p model.Project, role string) projectDTO {
	d := toProjectDTO(p)
	d.YourRole = role
	return d
}

type memberDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func toMemberDTO(m model.ProjectMember) memberDTO {
	return memberDTO{UserID: m.UserID.String(), Username: m.Username, Email: m.Email, Role: m.Role}
}

type adminUserDTO struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

func toAdminUserDTO(u model.User) adminUserDTO {
	return adminUserDTO{
		ID: u.ID.String(), Username: u.Username, Email: u.Email,
		IsAdmin: u.IsAdmin, IsActive: u.IsActive, CreatedAt: u.CreatedAt,
	}
}

type auditEntryDTO struct {
	ID         string    `json:"id"`
	ProjectID  *string   `json:"project_id"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type,omitempty"`
	TargetID   string    `json:"target_id,omitempty"`
	Status     int       `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func toAuditEntryDTO(e model.AuditEntry) auditEntryDTO {
	d := auditEntryDTO{
		ID: e.ID.String(), Actor: e.ActorUsername, Action: e.Action,
		TargetType: e.TargetType, TargetID: e.TargetID, Status: e.Status, CreatedAt: e.CreatedAt,
	}
	if e.ProjectID != nil {
		s := e.ProjectID.String()
		d.ProjectID = &s
	}
	return d
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

type patDTO struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func toPATDTO(p model.UserPAT) patDTO {
	return patDTO{
		ID: p.ID.String(), Name: p.Name, Prefix: p.Prefix,
		ExpiresAt: p.ExpiresAt, LastUsedAt: p.LastUsedAt, RevokedAt: p.RevokedAt, CreatedAt: p.CreatedAt,
	}
}

type versionMetaDTO struct {
	Version   int       `json:"version"`
	IsCurrent bool      `json:"is_current"`
	Kind      string    `json:"kind"`
	ByteSize  int64     `json:"byte_size"`
	Comment   string    `json:"comment,omitempty"`
	CreatedBy *string   `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func toVersionMetaDTO(v model.ConfigVersion) versionMetaDTO {
	d := versionMetaDTO{
		Version: v.Version, IsCurrent: v.IsCurrent, Kind: v.SnapshotKind,
		ByteSize: v.ByteSize, CreatedAt: v.CreatedAt,
	}
	if v.Comment != nil {
		d.Comment = *v.Comment
	}
	if v.CreatedBy != nil {
		s := v.CreatedBy.String()
		d.CreatedBy = &s
	}
	return d
}

type varDiffEntryDTO struct {
	Key       string  `json:"key"`
	Status    string  `json:"status"`
	From      *string `json:"from,omitempty"`
	To        *string `json:"to,omitempty"`
	WasSecret bool    `json:"was_secret"`
	IsSecret  bool    `json:"is_secret"`
}

type diffDTO struct {
	Kind     string            `json:"kind"`
	From     int               `json:"from"`
	To       int               `json:"to"`
	Vars     []varDiffEntryDTO `json:"vars,omitempty"`
	FileDiff string            `json:"file_diff,omitempty"`
}

func toDiffDTO(d app.DiffResult) diffDTO {
	out := diffDTO{Kind: d.Kind, From: d.From, To: d.To, FileDiff: d.FileDiff}
	for _, v := range d.Vars {
		out.Vars = append(out.Vars, varDiffEntryDTO{
			Key: v.Key, Status: v.Status, From: v.From, To: v.To, WasSecret: v.WasSecret, IsSecret: v.IsSecret,
		})
	}
	return out
}
