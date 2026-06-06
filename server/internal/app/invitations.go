package app

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

const invitationTTL = 7 * 24 * time.Hour

// CreateInvitation creates a pending invitation and returns it plus the raw
// token (shown once; the handler builds the accept link and sends/logs it).
func (s *Service) CreateInvitation(ctx context.Context, project model.Project, inviter model.User, email, role string) (model.Invitation, string, error) {
	if !validRole(role) {
		return model.Invitation{}, "", invalid("role", "must be owner, editor, or viewer")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return model.Invitation{}, "", invalid("email", "must be a valid email")
	}
	raw, err := newOpaqueToken()
	if err != nil {
		return model.Invitation{}, "", err
	}
	inviterID := inviter.ID
	inv, err := s.store.Q().CreateInvitation(ctx, project.ID, email, role, s.hashToken(raw), &inviterID, now().Add(invitationTTL))
	if err != nil {
		return model.Invitation{}, "", err
	}
	return inv, raw, nil
}

// ListInvitations returns a project's pending invitations.
func (s *Service) ListInvitations(ctx context.Context, projectID uuid.UUID) ([]model.Invitation, error) {
	return s.store.Q().ListPendingInvitations(ctx, projectID)
}

// RevokeInvitation deletes a pending invitation.
func (s *Service) RevokeInvitation(ctx context.Context, projectID, id uuid.UUID) error {
	return s.store.Q().DeleteInvitation(ctx, projectID, id)
}

// InvitationByToken returns a still-valid invitation and its project, for
// rendering the accept page. Expired/used/unknown tokens are ErrNotFound.
func (s *Service) InvitationByToken(ctx context.Context, rawToken string) (model.Invitation, model.Project, error) {
	inv, err := s.validInvitation(ctx, rawToken)
	if err != nil {
		return model.Invitation{}, model.Project{}, err
	}
	p, err := s.store.Q().GetProjectByID(ctx, inv.ProjectID)
	if err != nil {
		return model.Invitation{}, model.Project{}, err
	}
	return inv, p, nil
}

// AcceptInvitation validates the token and, in one transaction, creates a NEW
// user (the invitee sets their own username/password), adds the project
// membership, and marks the invitation accepted. Existing accounts must be
// added directly by an owner instead.
func (s *Service) AcceptInvitation(ctx context.Context, rawToken, username, password string) (model.User, error) {
	inv, err := s.validInvitation(ctx, rawToken)
	if err != nil {
		return model.User{}, err
	}
	if err := validateCredentials(username, inv.Email, password); err != nil {
		return model.User{}, err
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return model.User{}, err
	}
	var user model.User
	err = s.store.InTx(ctx, func(q *store.Queries) error {
		if _, err := q.GetUserByEmail(ctx, inv.Email); err == nil {
			return invalid("email", "an account already exists for this email — ask an owner to add you directly")
		} else if err != store.ErrNotFound {
			return err
		}
		u, err := q.CreateUser(ctx, model.User{Username: username, Email: inv.Email, PasswordHash: hash, IsAdmin: false})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrConflict // username taken
			}
			return err
		}
		if err := q.AddMember(ctx, inv.ProjectID, u.ID, inv.Role); err != nil {
			return err
		}
		if err := q.MarkInvitationAccepted(ctx, inv.ID, u.ID, now()); err != nil {
			return err
		}
		user = u
		return nil
	})
	if err != nil {
		return model.User{}, err
	}
	return user, nil
}

// validInvitation resolves a raw token to a pending, unexpired invitation.
func (s *Service) validInvitation(ctx context.Context, rawToken string) (model.Invitation, error) {
	inv, err := s.store.Q().GetInvitationByHash(ctx, s.hashToken(rawToken))
	if err != nil {
		if err == store.ErrNotFound {
			return model.Invitation{}, ErrNotFound
		}
		return model.Invitation{}, err
	}
	if inv.AcceptedAt != nil || inv.ExpiresAt.Before(now()) {
		return model.Invitation{}, ErrNotFound
	}
	return inv, nil
}
