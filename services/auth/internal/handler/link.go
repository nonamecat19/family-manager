package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/services/auth/db"
)

const (
	linkTokenTTL      = 10 * time.Minute
	maxExternalIDSize = 64
)

var linkProviders = map[string]struct{}{"telegram": {}}

func (h *Handler) CreateLinkToken(
	ctx context.Context, req *connect.Request[authv1.CreateLinkTokenRequest],
) (*connect.Response[authv1.CreateLinkTokenResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}

	provider, err := linkProvider(req.Msg.GetProvider())
	if err != nil {
		return nil, err
	}

	userID, err := pgconv.UUID(claims.UserID)
	if err != nil {
		return nil, invalid("malformed subject")
	}

	secret, err := newRefreshToken()
	if err != nil {
		return nil, h.internal(ctx, err, "generate link token")
	}

	if _, err := h.q.CreateLinkToken(ctx, db.CreateLinkTokenParams{
		UserID:    userID,
		Provider:  provider,
		TokenHash: hashToken(secret),
		ExpiresAt: pgconv.TimestampFrom(h.now().Add(linkTokenTTL)),
	}); err != nil {
		return nil, h.internal(ctx, err, "store link token")
	}

	return connect.NewResponse(&authv1.CreateLinkTokenResponse{
		Token:     secret,
		ExpiresIn: int64(linkTokenTTL.Seconds()),
	}), nil
}

func (h *Handler) RedeemLinkToken(
	ctx context.Context, req *connect.Request[authv1.RedeemLinkTokenRequest],
) (*connect.Response[authv1.RedeemLinkTokenResponse], error) {
	presented := strings.TrimSpace(req.Msg.GetToken())
	if presented == "" {
		return nil, invalid("token is required")
	}

	provider, err := linkProvider(req.Msg.GetProvider())
	if err != nil {
		return nil, err
	}

	externalID := strings.TrimSpace(req.Msg.GetExternalId())
	if externalID == "" {
		return nil, invalid("external_id is required")
	}
	if len(externalID) > maxExternalIDSize {
		return nil, invalid("external_id is too long")
	}

	row, err := h.q.GetLinkToken(ctx, hashToken(presented))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errInvalidLinkToken()
		}
		return nil, h.internal(ctx, err, "get link token")
	}

	if row.Provider != provider || row.UsedAt.Valid {
		return nil, errInvalidLinkToken()
	}
	if !row.ExpiresAt.Valid || !row.ExpiresAt.Time.After(h.now()) {
		return nil, errInvalidLinkToken()
	}

	spent, err := h.q.MarkLinkTokenUsed(ctx, db.MarkLinkTokenUsedParams{
		ID:         row.ID,
		ExternalID: &externalID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "mark link token used")
	}
	if spent == 0 {
		return nil, errInvalidLinkToken()
	}

	user, err := h.q.GetUserByID(ctx, row.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errInvalidLinkToken()
		}
		return nil, h.internal(ctx, err, "get user")
	}

	chainID, err := newChainID()
	if err != nil {
		return nil, h.internal(ctx, err, "generate chain id")
	}

	tokens, err := h.mintSession(ctx, user, chainID)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&authv1.RedeemLinkTokenResponse{
		UserId:       pgconv.UUIDString(user.ID),
		AccessToken:  tokens.access,
		RefreshToken: tokens.refresh,
		ExpiresIn:    tokens.expiresIn,
	}), nil
}

func linkProvider(raw string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := linkProviders[provider]; !ok {
		return "", invalid("unsupported provider")
	}
	return provider, nil
}

func errInvalidLinkToken() error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired link token"))
}
