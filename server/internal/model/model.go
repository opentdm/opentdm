// Package model holds opentdm's domain entities, shared by the store, service,
// and HTTP layers. They are plain data; behavior lives in the service layer.
package model

import (
	"time"

	"github.com/google/uuid"
)

// Config kinds and formats (mirror the DB enums).
const (
	KindVariable = "variable"
	KindFile     = "file"

	FormatEnv        = "env"
	FormatProperties = "properties"
	FormatSecret     = "secret"
	FormatJSON       = "json"
	FormatCSV        = "csv"
	FormatXML        = "xml"

	ScopeRead  = "read"
	ScopeWrite = "write"

	// Project member roles, ordered by privilege (compare with RoleRank).
	RoleViewer = "viewer"
	RoleEditor = "editor"
	RoleOwner  = "owner"
)

// RoleRank returns the privilege rank of a project role (higher = more), or 0
// for an unknown role. Used for "caller's role >= required role" checks.
func RoleRank(role string) int {
	switch role {
	case RoleViewer:
		return 1
	case RoleEditor:
		return 2
	case RoleOwner:
		return 3
	}
	return 0
}

type User struct {
	ID           uuid.UUID
	Username     string
	Email        string
	PasswordHash string
	IsAdmin      bool
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ProjectMember is a user's role on a project (Username/Email joined for display).
type ProjectMember struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Role      string
	Username  string
	Email     string
	CreatedAt time.Time
}

// AuditEntry is one recorded resource mutation (who did what, when).
type AuditEntry struct {
	ID            uuid.UUID
	ProjectID     *uuid.UUID
	ActorUserID   *uuid.UUID
	ActorUsername string // joined from users for display ("" if actor deleted)
	Action        string
	TargetType    string
	TargetID      string
	Status        int
	IP            string
	CreatedAt     time.Time
}

// Invitation is an email invitation to join a project with a role.
type Invitation struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	Email          string
	Role           string
	InvitedBy      *uuid.UUID
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	AcceptedUserID *uuid.UUID
	CreatedAt      time.Time
}

type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  []byte
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
}

type Project struct {
	ID            uuid.UUID
	Slug          string
	Name          string
	Description   string
	CreatedBy     *uuid.UUID
	DEKWrapped    []byte
	DEKKeyRef     string
	DEKVersion    int
	CryptoVersion int
	ArchivedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Environment struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Slug      string
	Name      string
	Rank      int
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Config struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Kind        string
	Format      string
	Name        string
	SortOrder   int
	Description string
	IsSecret    bool
	Tags        []string
	CreatedBy   *uuid.UUID
	ArchivedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ConfigItem is one variable at one layer. EnvID nil means the base layer.
type ConfigItem struct {
	ID              uuid.UUID
	ConfigID        uuid.UUID
	EnvID           *uuid.UUID
	Key             string
	ValueCiphertext []byte
	DEKVersion      int
	IsSecret        bool
	Deleted         bool
}

type Token struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	Name       string
	Prefix     string
	Scope      string
	EnvIDs     []uuid.UUID
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// ConfigBlob is the current content of a file config at one layer (EnvID nil =
// default variant).
type ConfigBlob struct {
	ID                uuid.UUID
	ConfigID          uuid.UUID
	EnvID             *uuid.UUID
	ContentCiphertext []byte
	DEKVersion        int
	ContentHMAC       []byte
	SizeBytes         int64
	UpdatedBy         *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ConfigVersion is one append-only snapshot of a config layer's whole content.
// For variables the decrypted snapshot is canonical JSON of the item set; for
// files it is the raw plaintext. ListVersions leaves SnapshotCiphertext nil.
type ConfigVersion struct {
	ID                 uuid.UUID
	ConfigID           uuid.UUID
	EnvID              *uuid.UUID
	Version            int
	SnapshotKind       string
	SnapshotCiphertext []byte
	DEKVersion         int
	ContentHMAC        []byte
	ByteSize           int64
	IsCurrent          bool
	Comment            *string
	CreatedBy          *uuid.UUID
	CreatedAt          time.Time
}

// UserPAT is a user-scoped Personal Access Token (authenticates as the user for
// management writes), distinct from project+environment service tokens.
type UserPAT struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Name       string
	Prefix     string
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}
