package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nnc/notes-manager-backend/internal/database/sqlc"
)

type Service struct {
	queries *sqlc.Queries
	jwt     *JWTManager
}

func NewService(queries *sqlc.Queries, jwt *JWTManager) *Service {
	return &Service{queries: queries, jwt: jwt}
}

func (s *Service) Register(ctx context.Context, email, username, password string) (*sqlc.User, *TokenPair, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:    email,
		Username: username,
		Password: hash,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	tokens, err := s.jwt.GenerateTokenPair(uuidToString(user.ID))
	if err != nil {
		return nil, nil, err
	}
	return &user, tokens, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*sqlc.User, *TokenPair, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}
	if !CheckPassword(user.Password, password) {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	tokens, err := s.jwt.GenerateTokenPair(uuidToString(user.ID))
	if err != nil {
		return nil, nil, err
	}
	return &user, tokens, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.jwt.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}
	return s.jwt.GenerateTokenPair(claims.UserID)
}

func uuidToString(u pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
