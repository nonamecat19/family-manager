package handler

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/services/auth/db"
)

func bindIdentity(
	ctx context.Context, q db.Querier,
	userID pgtype.UUID, provider, externalID string, chainID pgtype.UUID,
) error {
	if err := q.LockIdentityKey(ctx, identityKey(provider, externalID)); err != nil {
		return err
	}

	prev, err := q.GetIdentity(ctx, db.GetIdentityParams{Provider: provider, ExternalID: externalID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return err
	case prev.UserID != userID:
		return errAlreadyLinked()
	}

	if _, err := q.UpsertIdentity(ctx, db.UpsertIdentityParams{
		UserID:     userID,
		Provider:   provider,
		ExternalID: externalID,
		ChainID:    chainID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errAlreadyLinked()
		}
		return err
	}

	if prev.ChainID.Valid {
		if err := revokeChain(ctx, q, prev.ChainID); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) ListIdentities(
	ctx context.Context, _ *connect.Request[authv1.ListIdentitiesRequest],
) (*connect.Response[authv1.ListIdentitiesResponse], error) {
	userID, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := h.q.ListIdentitiesForUser(ctx, userID)
	if err != nil {
		return nil, h.internal(ctx, err, "list identities")
	}

	out := make([]*authv1.Identity, 0, len(rows))
	for _, r := range rows {
		out = append(out, &authv1.Identity{
			Provider:   r.Provider,
			ExternalId: r.ExternalID,
			LinkedAt:   pgconv.Timestamp(r.UpdatedAt),
		})
	}
	return connect.NewResponse(&authv1.ListIdentitiesResponse{Identities: out}), nil
}

func (h *Handler) Unlink(
	ctx context.Context, req *connect.Request[authv1.UnlinkRequest],
) (*connect.Response[authv1.UnlinkResponse], error) {
	userID, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}

	provider, err := linkProvider(req.Msg.GetProvider())
	if err != nil {
		return nil, err
	}
	externalID := strings.TrimSpace(req.Msg.GetExternalId())
	if externalID == "" {
		return nil, invalid("external_id is required")
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if err := q.LockIdentityKey(ctx, identityKey(provider, externalID)); err != nil {
			return err
		}
		gone, err := q.DeleteIdentity(ctx, db.DeleteIdentityParams{
			UserID: userID, Provider: provider, ExternalID: externalID,
		})
		if err != nil {
			return err
		}
		return revokeChain(ctx, q, gone.ChainID)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("identity is not linked"))
		}
		return nil, h.internal(ctx, err, "unlink identity")
	}
	return connect.NewResponse(&authv1.UnlinkResponse{}), nil
}

func identityKey(provider, externalID string) string {
	return provider + ":" + externalID
}

func revokeChain(ctx context.Context, q db.Querier, chainID pgtype.UUID) error {
	if err := q.TombstoneChain(ctx, chainID); err != nil {
		return err
	}
	_, err := q.RevokeChain(ctx, chainID)
	return err
}

func lockChain(ctx context.Context, q db.Querier, chainID pgtype.UUID, fresh bool) (pgtype.Timestamptz, error) {
	if fresh {
		return q.EnsureChain(ctx, chainID)
	}
	return q.LockChain(ctx, chainID)
}

func (h *Handler) caller(ctx context.Context) (pgtype.UUID, error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return pgtype.UUID{}, err
	}
	userID, err := pgconv.UUID(claims.UserID)
	if err != nil {
		return pgtype.UUID{}, invalid("malformed subject")
	}
	return userID, nil
}

func errAlreadyLinked() error {
	return connect.NewError(connect.CodeAlreadyExists,
		errors.New("this account is already linked to another user"))
}
