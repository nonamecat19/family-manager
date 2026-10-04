package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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

	callerChain, err := h.callerChain(ctx, req.Header())
	if err != nil {
		return nil, err
	}

	secret, err := newRefreshToken()
	if err != nil {
		return nil, h.internal(ctx, err, "generate link token")
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if err := requireLiveChain(ctx, q, callerChain); err != nil {
			return err
		}
		if err := requireFirstPartyChain(ctx, q, callerChain); err != nil {
			return err
		}
		_, err := q.CreateLinkToken(ctx, db.CreateLinkTokenParams{
			UserID:    userID,
			Provider:  provider,
			TokenHash: hashToken(secret),
			ExpiresAt: pgconv.TimestampFrom(h.now().Add(linkTokenTTL)),
		})
		return err
	})
	if err != nil {
		var cerr *connect.Error
		if errors.As(err, &cerr) {
			return nil, err
		}
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

	chainID, err := newChainID()
	if err != nil {
		return nil, h.internal(ctx, err, "generate chain id")
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		spent, err := q.MarkLinkTokenUsed(ctx, db.MarkLinkTokenUsedParams{
			ID:         row.ID,
			ExternalID: &externalID,
		})
		if err != nil {
			return err
		}
		if spent == 0 {
			return errInvalidLinkToken()
		}
		return bindIdentity(ctx, q, row.UserID, provider, externalID, chainID)
	})
	if err != nil {
		var cerr *connect.Error
		if errors.As(err, &cerr) {
			return nil, err
		}
		return nil, h.internal(ctx, err, "bind identity")
	}

	user, err := h.q.GetUserByID(ctx, row.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errInvalidLinkToken()
		}
		return nil, h.internal(ctx, err, "get user")
	}

	tokens, err := h.mintSession(ctx, user, chainID, true)
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

func requireLiveChain(ctx context.Context, q db.Querier, chainID pgtype.UUID) error {
	revokedAt, err := q.LockChain(ctx, chainID)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && revokedAt.Valid {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("this session has been signed out"))
	}
	return err
}

func requireFirstPartyChain(ctx context.Context, q db.Querier, chainID pgtype.UUID) error {
	root, err := q.GetChainRoot(ctx, chainID)
	if err != nil {
		return err
	}
	for _, c := range []pgtype.UUID{chainID, root} {
		linked, err := q.IsIdentityChain(ctx, c)
		if err != nil {
			return err
		}
		if linked {
			return connect.NewError(connect.CodePermissionDenied,
				errors.New("connect accounts from a session you signed in to with your password"))
		}
	}
	return nil
}
