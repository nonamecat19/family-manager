package handler

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nnc/finance-tracker/server/internal/db/sqlc"
)

// PgFamilyDB implements FamilyDB using sqlc-generated queries against PostgreSQL.
type PgFamilyDB struct {
	queries *sqlc.Queries
}

// NewPgFamilyDB creates a PgFamilyDB wrapping sqlc.Queries.
func NewPgFamilyDB(queries *sqlc.Queries) *PgFamilyDB {
	return &PgFamilyDB{queries: queries}
}

func (db *PgFamilyDB) CreateFamily(userID, name string) (MockFamily, error) {
	uid := stringToUUID(userID)
	row, err := db.queries.CreateFamily(context.Background(), sqlc.CreateFamilyParams{
		Name:        name,
		AdminUserID: uid,
	})
	if err != nil {
		return MockFamily{}, err
	}
	return MockFamily{
		ID:          uuidToString(row.ID),
		Name:        row.Name,
		AdminUserID: uuidToString(row.AdminUserID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (db *PgFamilyDB) GetFamilyByUserID(userID string) (MockFamily, error) {
	uid := stringToUUID(userID)
	row, err := db.queries.GetFamilyByUserID(context.Background(), uid)
	if err != nil {
		return MockFamily{}, ErrFamilyNotFound
	}
	return MockFamily{
		ID:          uuidToString(row.ID),
		Name:        row.Name,
		AdminUserID: uuidToString(row.AdminUserID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (db *PgFamilyDB) GetFamilyMembers(familyID string) ([]MockFamilyMember, error) {
	fid := stringToUUID(familyID)
	rows, err := db.queries.GetFamilyMembers(context.Background(), fid)
	if err != nil {
		return nil, err
	}
	members := make([]MockFamilyMember, len(rows))
	for i, row := range rows {
		members[i] = MockFamilyMember{
			ID:       uuidToString(row.ID),
			UserID:   uuidToString(row.UserID),
			Email:    row.Email,
			Role:     row.Role,
			JoinedAt: row.JoinedAt.Time,
		}
	}
	return members, nil
}

func (db *PgFamilyDB) AddFamilyMember(familyID, userID, role string) error {
	fid := stringToUUID(familyID)
	uid := stringToUUID(userID)
	_, err := db.queries.AddFamilyMember(context.Background(), sqlc.AddFamilyMemberParams{
		FamilyID: fid,
		UserID:   uid,
		Role:     role,
	})
	return err
}

func (db *PgFamilyDB) RemoveFamilyMember(familyID, userID string) (int64, error) {
	fid := stringToUUID(familyID)
	uid := stringToUUID(userID)
	return db.queries.RemoveFamilyMember(context.Background(), sqlc.RemoveFamilyMemberParams{
		FamilyID: fid,
		UserID:   uid,
	})
}

func (db *PgFamilyDB) DeleteFamily(familyID, adminUserID string) (int64, error) {
	fid := stringToUUID(familyID)
	uid := stringToUUID(adminUserID)
	return db.queries.DeleteFamily(context.Background(), sqlc.DeleteFamilyParams{
		ID:          fid,
		AdminUserID: uid,
	})
}

func (db *PgFamilyDB) GetFamilyMemberCount(familyID string) (int64, error) {
	fid := stringToUUID(familyID)
	return db.queries.GetFamilyMemberCount(context.Background(), fid)
}

func (db *PgFamilyDB) CreateInvitation(familyID, inviterUserID, tokenHash string, expiresAt time.Time) (MockInvitation, error) {
	fid := stringToUUID(familyID)
	uid := stringToUUID(inviterUserID)
	row, err := db.queries.CreateInvitation(context.Background(), sqlc.CreateInvitationParams{
		FamilyID:      fid,
		InviterUserID: uid,
		TokenHash:     tokenHash,
		ExpiresAt:     pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return MockInvitation{}, err
	}
	return MockInvitation{
		ID:            uuidToString(row.ID),
		FamilyID:      uuidToString(row.FamilyID),
		InviterUserID: uuidToString(row.InviterUserID),
		TokenHash:     row.TokenHash,
		Status:        row.Status,
		ExpiresAt:     row.ExpiresAt.Time,
		CreatedAt:     row.CreatedAt.Time,
	}, nil
}

func (db *PgFamilyDB) GetInvitationByTokenHash(tokenHash string) (MockInvitation, error) {
	row, err := db.queries.GetInvitationByTokenHash(context.Background(), tokenHash)
	if err != nil {
		return MockInvitation{}, ErrInvitationNotFound
	}
	return MockInvitation{
		ID:            uuidToString(row.ID),
		FamilyID:      uuidToString(row.FamilyID),
		InviterUserID: uuidToString(row.InviterUserID),
		TokenHash:     row.TokenHash,
		Status:        row.Status,
		FamilyName:    row.FamilyName,
		ExpiresAt:     row.ExpiresAt.Time,
		CreatedAt:     row.CreatedAt.Time,
	}, nil
}

func (db *PgFamilyDB) AcceptInvitation(invitationID string) (int64, error) {
	iid := stringToUUID(invitationID)
	return db.queries.AcceptInvitation(context.Background(), iid)
}

func (db *PgFamilyDB) RevokeInvitation(invitationID, familyID string) (int64, error) {
	iid := stringToUUID(invitationID)
	fid := stringToUUID(familyID)
	return db.queries.RevokeInvitation(context.Background(), sqlc.RevokeInvitationParams{
		ID:       iid,
		FamilyID: fid,
	})
}

func (db *PgFamilyDB) GetPendingInvitations(familyID string) ([]MockPendingInvitation, error) {
	fid := stringToUUID(familyID)
	rows, err := db.queries.GetPendingInvitations(context.Background(), fid)
	if err != nil {
		return nil, err
	}
	invitations := make([]MockPendingInvitation, len(rows))
	for i, row := range rows {
		invitations[i] = MockPendingInvitation{
			ID:            uuidToString(row.ID),
			FamilyID:      uuidToString(row.FamilyID),
			InviterUserID: uuidToString(row.InviterUserID),
			Status:        row.Status,
			ExpiresAt:     row.ExpiresAt.Time,
			CreatedAt:     row.CreatedAt.Time,
		}
	}
	return invitations, nil
}
