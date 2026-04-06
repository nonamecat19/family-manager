package handler

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nnc/finance-tracker/server/internal/db/sqlc"
)

// PgAuthDB implements AuthDB using sqlc-generated queries against PostgreSQL.
type PgAuthDB struct {
	queries *sqlc.Queries
}

// NewPgAuthDB creates a PgAuthDB wrapping sqlc.Queries.
func NewPgAuthDB(queries *sqlc.Queries) *PgAuthDB {
	return &PgAuthDB{queries: queries}
}

func (db *PgAuthDB) CreateUser(email, passwordHash string) (MockUser, error) {
	row, err := db.queries.CreateUser(context.Background(), sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return MockUser{}, ErrDuplicateEmail
		}
		return MockUser{}, err
	}

	uid := uuidToString(row.ID)
	return MockUser{
		ID:    uid,
		Email: row.Email,
	}, nil
}

func (db *PgAuthDB) GetUserByEmail(email string) (MockUser, error) {
	row, err := db.queries.GetUserByEmail(context.Background(), email)
	if err != nil {
		return MockUser{}, ErrUserNotFound
	}

	uid := uuidToString(row.ID)
	return MockUser{
		ID:           uid,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
	}, nil
}

func (db *PgAuthDB) StoreRefreshToken(userID, tokenHash string, expiresInDays int) error {
	uid := stringToUUID(userID)
	expiresAt := pgtype.Timestamptz{
		Time:  time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour),
		Valid: true,
	}
	_, err := db.queries.CreateRefreshToken(context.Background(), sqlc.CreateRefreshTokenParams{
		UserID:    uid,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	return err
}

func (db *PgAuthDB) GetRefreshTokenByHash(tokenHash string) (MockRefreshToken, error) {
	row, err := db.queries.GetRefreshTokenByHash(context.Background(), tokenHash)
	if err != nil {
		return MockRefreshToken{}, ErrTokenNotFound
	}

	uid := uuidToString(row.UserID)
	return MockRefreshToken{
		UserID:    uid,
		TokenHash: row.TokenHash,
	}, nil
}

func (db *PgAuthDB) RevokeRefreshToken(tokenHash string) error {
	return db.queries.RevokeRefreshToken(context.Background(), tokenHash)
}

// uuidToString converts a pgtype.UUID to its string representation.
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return formatUUID(b)
}

// stringToUUID parses a UUID string into pgtype.UUID.
func stringToUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

// formatUUID formats a [16]byte as a UUID string.
func formatUUID(b [16]byte) string {
	return uuidHex(b[0:4]) + "-" + uuidHex(b[4:6]) + "-" + uuidHex(b[6:8]) + "-" + uuidHex(b[8:10]) + "-" + uuidHex(b[10:16])
}

func uuidHex(b []byte) string {
	const hexDigits = "0123456789abcdef"
	s := make([]byte, len(b)*2)
	for i, v := range b {
		s[i*2] = hexDigits[v>>4]
		s[i*2+1] = hexDigits[v&0x0f]
	}
	return string(s)
}
