package graph

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nnc/notes-manager-backend/internal/database/sqlc"
	"github.com/nnc/notes-manager-backend/internal/graph/model"
)

func uuidToString(u pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

func stringToUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func tsToTime(ts pgtype.Timestamptz) time.Time {
	return ts.Time
}

func userToModel(u sqlc.User) *model.User {
	return &model.User{
		ID:        uuidToString(u.ID),
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: tsToTime(u.CreatedAt),
		UpdatedAt: tsToTime(u.UpdatedAt),
	}
}

func listToModel(l sqlc.List) *model.List {
	return &model.List{
		ID:          uuidToString(l.ID),
		Title:       l.Title,
		Description: l.Description,
		IsPublic:    l.IsPublic,
		CreatedAt:   tsToTime(l.CreatedAt),
		UpdatedAt:   tsToTime(l.UpdatedAt),
	}
}

func itemToModel(i sqlc.Item) *model.Item {
	return &model.Item{
		ID:        uuidToString(i.ID),
		Type:      dbItemTypeToModel(i.Type),
		Content:   i.Content,
		Position:  int(i.Position),
		CreatedAt: tsToTime(i.CreatedAt),
		UpdatedAt: tsToTime(i.UpdatedAt),
	}
}

func tagToModel(t sqlc.Tag) *model.Tag {
	return &model.Tag{
		ID:        uuidToString(t.ID),
		Name:      t.Name,
		CreatedAt: tsToTime(t.CreatedAt),
	}
}

func dbItemTypeToModel(t sqlc.ItemType) model.ItemType {
	switch t {
	case sqlc.ItemTypeImage:
		return model.ItemTypeImage
	case sqlc.ItemTypeLink:
		return model.ItemTypeLink
	default:
		return model.ItemTypeText
	}
}

func modelItemTypeToDb(t model.ItemType) sqlc.ItemType {
	switch t {
	case model.ItemTypeImage:
		return sqlc.ItemTypeImage
	case model.ItemTypeLink:
		return sqlc.ItemTypeLink
	default:
		return sqlc.ItemTypeText
	}
}
