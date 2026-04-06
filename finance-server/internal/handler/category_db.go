package handler

import (
	"context"

	"github.com/nnc/finance-tracker/server/internal/db/sqlc"
)

// PgCategoryDB implements CategoryDB using sqlc-generated queries against PostgreSQL.
type PgCategoryDB struct {
	queries *sqlc.Queries
}

// NewPgCategoryDB creates a PgCategoryDB wrapping sqlc.Queries.
func NewPgCategoryDB(queries *sqlc.Queries) *PgCategoryDB {
	return &PgCategoryDB{queries: queries}
}

func (db *PgCategoryDB) CreateCategory(userID, name, icon, color string) (MockCategory, error) {
	uid := stringToUUID(userID)
	row, err := db.queries.CreateCategory(context.Background(), sqlc.CreateCategoryParams{
		UserID: uid,
		Name:   name,
		Icon:   icon,
		Color:  color,
	})
	if err != nil {
		return MockCategory{}, err
	}

	return MockCategory{
		ID:        uuidToString(row.ID),
		UserID:    uuidToString(row.UserID),
		Name:      row.Name,
		Icon:      row.Icon,
		Color:     row.Color,
		SortOrder: int(row.SortOrder),
	}, nil
}

func (db *PgCategoryDB) GetCategoriesByUser(userID string) ([]MockCategory, error) {
	uid := stringToUUID(userID)
	rows, err := db.queries.GetCategoriesByUser(context.Background(), uid)
	if err != nil {
		return nil, err
	}

	cats := make([]MockCategory, len(rows))
	for i, row := range rows {
		cats[i] = MockCategory{
			ID:        uuidToString(row.ID),
			UserID:    uuidToString(row.UserID),
			Name:      row.Name,
			Icon:      row.Icon,
			Color:     row.Color,
			SortOrder: int(row.SortOrder),
		}
	}
	return cats, nil
}

func (db *PgCategoryDB) GetCategoryByID(id, userID string) (MockCategory, error) {
	cid := stringToUUID(id)
	uid := stringToUUID(userID)
	row, err := db.queries.GetCategoryByID(context.Background(), sqlc.GetCategoryByIDParams{
		ID:     cid,
		UserID: uid,
	})
	if err != nil {
		return MockCategory{}, ErrCategoryNotFound
	}

	return MockCategory{
		ID:        uuidToString(row.ID),
		UserID:    uuidToString(row.UserID),
		Name:      row.Name,
		Icon:      row.Icon,
		Color:     row.Color,
		SortOrder: int(row.SortOrder),
	}, nil
}

func (db *PgCategoryDB) UpdateCategory(id, userID, name, icon, color string) error {
	cid := stringToUUID(id)
	uid := stringToUUID(userID)
	rowsAffected, err := db.queries.UpdateCategory(context.Background(), sqlc.UpdateCategoryParams{
		ID:     cid,
		UserID: uid,
		Name:   name,
		Icon:   icon,
		Color:  color,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (db *PgCategoryDB) DeleteCategory(id, userID string) error {
	cid := stringToUUID(id)
	uid := stringToUUID(userID)
	rowsAffected, err := db.queries.DeleteCategory(context.Background(), sqlc.DeleteCategoryParams{
		ID:     cid,
		UserID: uid,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (db *PgCategoryDB) UpdateCategorySortOrder(id, userID string, sortOrder int) error {
	cid := stringToUUID(id)
	uid := stringToUUID(userID)
	return db.queries.UpdateCategorySortOrder(context.Background(), sqlc.UpdateCategorySortOrderParams{
		ID:        cid,
		UserID:    uid,
		SortOrder: int32(sortOrder),
	})
}
