package handler

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/services/auth/db"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

// fakeStore is an in-memory db.Querier. Rotation and replay are handler behaviour, so they
// are what these tests exercise; Postgres is not under test.
type fakeStore struct {
	users  map[string]db.User         // keyed by lowercased email
	byID   map[string]db.User         // keyed by uuid string
	tokens map[string]db.RefreshToken // keyed by token hash

	failOn map[string]error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:  map[string]db.User{},
		byID:   map[string]db.User{},
		tokens: map[string]db.RefreshToken{},
		failOn: map[string]error{},
	}
}

func (s *fakeStore) fail(op string) error { return s.failOn[op] }

// uniqueViolation mimics Postgres SQLSTATE 23505 so the handler's mapping is exercised.
type uniqueViolation struct{}

func (uniqueViolation) Error() string    { return "duplicate key value violates unique constraint" }
func (uniqueViolation) SQLState() string { return "23505" }

func (s *fakeStore) CreateUser(_ context.Context, arg db.CreateUserParams) (db.User, error) {
	if err := s.fail("CreateUser"); err != nil {
		return db.User{}, err
	}
	if _, exists := s.users[arg.Email]; exists {
		return db.User{}, uniqueViolation{}
	}
	u := db.User{
		ID:           pgconv.MustUUID(newUUID()),
		Email:        arg.Email,
		Name:         arg.Name,
		PasswordHash: arg.PasswordHash,
		CreatedAt:    pgtype.Timestamptz{Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Valid: true},
	}
	s.users[arg.Email] = u
	s.byID[pgconv.UUIDString(u.ID)] = u
	return u, nil
}

func (s *fakeStore) GetUserByEmail(_ context.Context, email string) (db.User, error) {
	u, ok := s.users[email]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (s *fakeStore) GetUserByID(_ context.Context, id pgtype.UUID) (db.User, error) {
	u, ok := s.byID[pgconv.UUIDString(id)]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (s *fakeStore) CreateRefreshToken(
	_ context.Context, arg db.CreateRefreshTokenParams,
) (db.RefreshToken, error) {
	if err := s.fail("CreateRefreshToken"); err != nil {
		return db.RefreshToken{}, err
	}
	// Mirrors the query's WHERE NOT EXISTS: a revoked chain accepts no successor.
	for _, existing := range s.tokens {
		if pgconv.UUIDString(existing.ChainID) == pgconv.UUIDString(arg.ChainID) &&
			existing.RevokedAt.Valid {
			return db.RefreshToken{}, pgx.ErrNoRows
		}
	}
	t := db.RefreshToken{
		ID:        pgconv.MustUUID(newUUID()),
		UserID:    arg.UserID,
		TokenHash: arg.TokenHash,
		ChainID:   arg.ChainID,
		ExpiresAt: arg.ExpiresAt,
		CreatedAt: pgtype.Timestamptz{Valid: true},
	}
	s.tokens[arg.TokenHash] = t
	return t, nil
}

func (s *fakeStore) GetRefreshToken(_ context.Context, hash string) (db.RefreshToken, error) {
	t, ok := s.tokens[hash]
	if !ok {
		return db.RefreshToken{}, pgx.ErrNoRows
	}
	return t, nil
}

func (s *fakeStore) MarkRefreshTokenUsed(_ context.Context, id pgtype.UUID) (int64, error) {
	for hash, t := range s.tokens {
		if pgconv.UUIDString(t.ID) != pgconv.UUIDString(id) {
			continue
		}
		// Mirrors the query's WHERE: only an unused, unrevoked row updates.
		if t.UsedAt.Valid || t.RevokedAt.Valid {
			return 0, nil
		}
		t.UsedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		s.tokens[hash] = t
		return 1, nil
	}
	return 0, nil
}

func (s *fakeStore) RevokeChain(_ context.Context, chainID pgtype.UUID) (int64, error) {
	var n int64
	for hash, t := range s.tokens {
		if pgconv.UUIDString(t.ChainID) == pgconv.UUIDString(chainID) && !t.RevokedAt.Valid {
			t.RevokedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
			s.tokens[hash] = t
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) RevokeAllForUser(_ context.Context, userID pgtype.UUID) (int64, error) {
	var n int64
	for hash, t := range s.tokens {
		if pgconv.UUIDString(t.UserID) == pgconv.UUIDString(userID) && !t.RevokedAt.Valid {
			t.RevokedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
			s.tokens[hash] = t
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) DeleteExpiredRefreshTokens(context.Context) (int64, error) { return 0, nil }

// stubSigner records what it was asked to sign, so tests can assert on the claims without
// parsing a JWT — token_test.go already covers the real signing path.
type stubSigner struct {
	signed []token.Claims
	err    error
	serial int
}

func (s *stubSigner) Sign(c token.Claims) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	s.signed = append(s.signed, c)
	s.serial++
	return "access-token-" + string(rune('a'+s.serial-1)), nil
}

func (s *stubSigner) TTL() time.Duration { return 15 * time.Minute }

func (s *stubSigner) last() token.Claims {
	if len(s.signed) == 0 {
		return token.Claims{}
	}
	return s.signed[len(s.signed)-1]
}

// stubFamily stands in for services/family's internal listener.
type stubFamily struct {
	familyID string
	err      error
	calls    int
}

func (f *stubFamily) FamilyOf(context.Context, string) (string, error) {
	f.calls++
	return f.familyID, f.err
}

var errBoom = errors.New("boom")
