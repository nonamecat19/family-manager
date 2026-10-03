package handler

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/services/auth/db"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

type fakeStore struct {
	users  map[string]db.User
	byID   map[string]db.User
	tokens map[string]db.RefreshToken
	links  map[string]db.LinkToken
	idents map[string]db.Identity
	chains map[string]pgtype.Timestamptz
	grants map[string]db.LoginGrant

	failOn map[string]error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:  map[string]db.User{},
		byID:   map[string]db.User{},
		tokens: map[string]db.RefreshToken{},
		links:  map[string]db.LinkToken{},
		idents: map[string]db.Identity{},
		chains: map[string]pgtype.Timestamptz{},
		grants: map[string]db.LoginGrant{},
		failOn: map[string]error{},
	}
}

func (s *fakeStore) fail(op string) error { return s.failOn[op] }

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
		ID:           mustChainID(),
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
	for _, existing := range s.tokens {
		if pgconv.UUIDString(existing.ChainID) == pgconv.UUIDString(arg.ChainID) &&
			existing.RevokedAt.Valid {
			return db.RefreshToken{}, pgx.ErrNoRows
		}
	}
	t := db.RefreshToken{
		ID:        mustChainID(),
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

func (s *fakeStore) CreateLinkToken(
	_ context.Context, arg db.CreateLinkTokenParams,
) (db.LinkToken, error) {
	if err := s.fail("CreateLinkToken"); err != nil {
		return db.LinkToken{}, err
	}
	t := db.LinkToken{
		ID:        mustChainID(),
		UserID:    arg.UserID,
		Provider:  arg.Provider,
		TokenHash: arg.TokenHash,
		ExpiresAt: arg.ExpiresAt,
		CreatedAt: pgtype.Timestamptz{Valid: true},
	}
	s.links[arg.TokenHash] = t
	return t, nil
}

func (s *fakeStore) GetLinkToken(_ context.Context, hash string) (db.LinkToken, error) {
	t, ok := s.links[hash]
	if !ok {
		return db.LinkToken{}, pgx.ErrNoRows
	}
	return t, nil
}

func (s *fakeStore) MarkLinkTokenUsed(
	_ context.Context, arg db.MarkLinkTokenUsedParams,
) (int64, error) {
	for hash, t := range s.links {
		if pgconv.UUIDString(t.ID) != pgconv.UUIDString(arg.ID) {
			continue
		}
		if t.UsedAt.Valid {
			return 0, nil
		}
		t.UsedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		t.ExternalID = arg.ExternalID
		s.links[hash] = t
		return 1, nil
	}
	return 0, nil
}

func (s *fakeStore) DeleteExpiredLinkTokens(context.Context) (int64, error) { return 0, nil }

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

type stubFamily struct {
	familyID string
	locale   string
	err      error
	calls    int
}

func (f *stubFamily) FamilyOf(context.Context, string) (string, error) {
	f.calls++
	return f.familyID, f.err
}

func (f *stubFamily) LocaleOf(context.Context, string) (string, error) {
	return f.locale, f.err
}

var errBoom = errors.New("boom")

func mustChainID() pgtype.UUID {
	id, err := newChainID()
	if err != nil {
		panic(err)
	}
	return id
}

func (s *fakeStore) InTx(_ context.Context, fn func(db.Querier) error) error {
	tokens, links, idents := maps.Clone(s.tokens), maps.Clone(s.links), maps.Clone(s.idents)
	chains := maps.Clone(s.chains)
	if err := fn(s); err != nil {
		s.tokens, s.links, s.idents, s.chains = tokens, links, idents, chains
		return err
	}
	return nil
}

func identKey(provider, externalID string) string { return provider + "|" + externalID }

func (s *fakeStore) GetIdentity(_ context.Context, arg db.GetIdentityParams) (db.Identity, error) {
	if err := s.fail("GetIdentity"); err != nil {
		return db.Identity{}, err
	}
	i, ok := s.idents[identKey(arg.Provider, arg.ExternalID)]
	if !ok {
		return db.Identity{}, pgx.ErrNoRows
	}
	return i, nil
}

func (s *fakeStore) UpsertIdentity(_ context.Context, arg db.UpsertIdentityParams) (db.Identity, error) {
	if err := s.fail("UpsertIdentity"); err != nil {
		return db.Identity{}, err
	}
	key := identKey(arg.Provider, arg.ExternalID)
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	if i, ok := s.idents[key]; ok {
		if i.UserID != arg.UserID {
			return db.Identity{}, pgx.ErrNoRows
		}
		i.ChainID, i.UpdatedAt = arg.ChainID, now
		s.idents[key] = i
		return i, nil
	}
	i := db.Identity{
		ID:         mustChainID(),
		UserID:     arg.UserID,
		Provider:   arg.Provider,
		ExternalID: arg.ExternalID,
		ChainID:    arg.ChainID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.idents[key] = i
	return i, nil
}

func (s *fakeStore) ListIdentitiesForUser(_ context.Context, userID pgtype.UUID) ([]db.Identity, error) {
	if err := s.fail("ListIdentitiesForUser"); err != nil {
		return nil, err
	}
	var out []db.Identity
	for _, i := range s.idents {
		if i.UserID == userID {
			out = append(out, i)
		}
	}
	return out, nil
}

func (s *fakeStore) DeleteIdentity(_ context.Context, arg db.DeleteIdentityParams) (db.Identity, error) {
	if err := s.fail("DeleteIdentity"); err != nil {
		return db.Identity{}, err
	}
	key := identKey(arg.Provider, arg.ExternalID)
	i, ok := s.idents[key]
	if !ok || i.UserID != arg.UserID {
		return db.Identity{}, pgx.ErrNoRows
	}
	delete(s.idents, key)
	return i, nil
}

func (s *fakeStore) LockIdentityKey(context.Context, string) error { return nil }

func (s *fakeStore) EnsureChain(_ context.Context, id pgtype.UUID) (pgtype.Timestamptz, error) {
	if err := s.fail("EnsureChain"); err != nil {
		return pgtype.Timestamptz{}, err
	}
	key := pgconv.UUIDString(id)
	revokedAt := s.chains[key]
	s.chains[key] = revokedAt
	return revokedAt, nil
}

func (s *fakeStore) TombstoneChain(_ context.Context, id pgtype.UUID) error {
	if err := s.fail("TombstoneChain"); err != nil {
		return err
	}
	key := pgconv.UUIDString(id)
	if !s.chains[key].Valid {
		s.chains[key] = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}
	return nil
}

func (s *fakeStore) DeleteOrphanChains(context.Context) (int64, error) { return 0, nil }

func (s *fakeStore) LockChain(_ context.Context, id pgtype.UUID) (pgtype.Timestamptz, error) {
	if err := s.fail("LockChain"); err != nil {
		return pgtype.Timestamptz{}, err
	}
	revokedAt, ok := s.chains[pgconv.UUIDString(id)]
	if !ok {
		return pgtype.Timestamptz{}, pgx.ErrNoRows
	}
	return revokedAt, nil
}

func (s *fakeStore) CreateLoginGrant(_ context.Context, arg db.CreateLoginGrantParams) (db.LoginGrant, error) {
	if err := s.fail("CreateLoginGrant"); err != nil {
		return db.LoginGrant{}, err
	}
	for _, g := range s.grants {
		if g.UserCodeHash == arg.UserCodeHash || g.DeviceCodeHash == arg.DeviceCodeHash {
			return db.LoginGrant{}, uniqueViolation{}
		}
	}
	g := db.LoginGrant{
		ID:             mustChainID(),
		Kind:           arg.Kind,
		DeviceCodeHash: arg.DeviceCodeHash,
		UserCodeHash:   arg.UserCodeHash,
		ExpiresAt:      arg.ExpiresAt,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	s.grants[pgconv.UUIDString(g.ID)] = g
	return g, nil
}

func (s *fakeStore) GetLoginGrantByDeviceCode(_ context.Context, hash string) (db.LoginGrant, error) {
	for _, g := range s.grants {
		if g.DeviceCodeHash == hash {
			return g, nil
		}
	}
	return db.LoginGrant{}, pgx.ErrNoRows
}

func (s *fakeStore) GetLoginGrantByUserCode(_ context.Context, hash string) (db.LoginGrant, error) {
	for _, g := range s.grants {
		if g.UserCodeHash == hash {
			return g, nil
		}
	}
	return db.LoginGrant{}, pgx.ErrNoRows
}

func (s *fakeStore) TouchLoginGrant(_ context.Context, arg db.TouchLoginGrantParams) (int64, error) {
	key := pgconv.UUIDString(arg.ID)
	g, ok := s.grants[key]
	if !ok || g.LastPolledAt.Valid && g.LastPolledAt.Time.After(arg.NotAfter.Time) {
		return 0, nil
	}
	g.LastPolledAt = arg.PolledAt
	s.grants[key] = g
	return 1, nil
}

func (s *fakeStore) decidable(hash string, at pgtype.Timestamptz) (string, db.LoginGrant, bool) {
	for key, g := range s.grants {
		if g.UserCodeHash == hash && !g.ApprovedAt.Valid && !g.DeniedAt.Valid &&
			g.ExpiresAt.Time.After(at.Time) {
			return key, g, true
		}
	}
	return "", db.LoginGrant{}, false
}

func (s *fakeStore) ApproveLoginGrant(_ context.Context, arg db.ApproveLoginGrantParams) (db.LoginGrant, error) {
	key, g, ok := s.decidable(arg.UserCodeHash, arg.DecidedAt)
	if !ok {
		return db.LoginGrant{}, pgx.ErrNoRows
	}
	g.UserID, g.ApprovedAt = arg.UserID, arg.DecidedAt
	s.grants[key] = g
	return g, nil
}

func (s *fakeStore) DenyLoginGrant(_ context.Context, arg db.DenyLoginGrantParams) (int64, error) {
	key, g, ok := s.decidable(arg.UserCodeHash, arg.DecidedAt)
	if !ok {
		return 0, nil
	}
	g.UserID, g.DeniedAt = arg.UserID, arg.DecidedAt
	s.grants[key] = g
	return 1, nil
}

func (s *fakeStore) ConsumeLoginGrant(_ context.Context, id pgtype.UUID) (int64, error) {
	key := pgconv.UUIDString(id)
	g, ok := s.grants[key]
	if !ok || !g.ApprovedAt.Valid || g.DeniedAt.Valid || g.ConsumedAt.Valid {
		return 0, nil
	}
	g.ConsumedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	s.grants[key] = g
	return 1, nil
}

func (s *fakeStore) DeleteExpiredLoginGrants(context.Context) (int64, error) { return 0, nil }
