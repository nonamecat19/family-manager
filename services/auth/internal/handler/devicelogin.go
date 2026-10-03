package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/services/auth/db"
)

const (
	deviceLoginTTL      = 10 * time.Minute
	deviceLoginInterval = 5 * time.Second
	deviceLoginSlowDown = 5 * time.Second
	devicePollSlack     = time.Second
	userCodeAlphabet    = "BCDFGHJKLMNPQRSTVWXZ"
	userCodeLength      = 8
	userCodeAttempts    = 3
	telegramLoginPrefix = "login_"
)

var deviceLoginKinds = map[authv1.DeviceLoginKind]string{
	authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE:   "device",
	authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM: "telegram",
}

func (h *Handler) StartDeviceLogin(
	ctx context.Context, req *connect.Request[authv1.StartDeviceLoginRequest],
) (*connect.Response[authv1.StartDeviceLoginResponse], error) {
	kind := req.Msg.GetKind()
	if kind == authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_UNSPECIFIED {
		kind = authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE
	}
	kindName, ok := deviceLoginKinds[kind]
	if !ok {
		return nil, invalid("unsupported kind")
	}

	deviceCode, err := newRefreshToken()
	if err != nil {
		return nil, h.internal(ctx, err, "generate device code")
	}
	expiresAt := h.now().Add(deviceLoginTTL)

	var userCode string
	for attempt := 1; ; attempt++ {
		userCode, err = newUserCode()
		if err != nil {
			return nil, h.internal(ctx, err, "generate user code")
		}
		_, err = h.q.CreateLoginGrant(ctx, db.CreateLoginGrantParams{
			Kind:           kindName,
			DeviceCodeHash: hashToken(deviceCode),
			UserCodeHash:   hashToken(userCode),
			ExpiresAt:      pgconv.TimestampFrom(expiresAt),
		})
		if err == nil {
			break
		}
		if !isUniqueViolation(err) || attempt == userCodeAttempts {
			return nil, h.internal(ctx, err, "store login grant")
		}
	}

	res := &authv1.StartDeviceLoginResponse{
		DeviceCode:      deviceCode,
		UserCode:        formatUserCode(userCode),
		ExpiresAt:       timestamppb.New(expiresAt),
		IntervalSeconds: int32(deviceLoginInterval.Seconds()),
	}
	if kind == authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM {
		res.TelegramStartPayload = telegramLoginPrefix + userCode
	}
	return connect.NewResponse(res), nil
}

func (h *Handler) PollDeviceLogin(
	ctx context.Context, req *connect.Request[authv1.PollDeviceLoginRequest],
) (*connect.Response[authv1.PollDeviceLoginResponse], error) {
	presented := strings.TrimSpace(req.Msg.GetDeviceCode())
	if presented == "" {
		return nil, invalid("device_code is required")
	}

	row, err := h.q.GetLoginGrantByDeviceCode(ctx, hashToken(presented))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pollStatus(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED, 0), nil
		}
		return nil, h.internal(ctx, err, "get login grant")
	}

	now := h.now()
	switch {
	case row.ConsumedAt.Valid, !row.ExpiresAt.Valid, !row.ExpiresAt.Time.After(now):
		return pollStatus(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED, 0), nil
	case row.DeniedAt.Valid:
		return pollStatus(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_DENIED, 0), nil
	case row.ApprovedAt.Valid:
		return h.redeemGrant(ctx, row)
	}

	touched, err := h.q.TouchLoginGrant(ctx, db.TouchLoginGrantParams{
		PolledAt: pgconv.TimestampFrom(now),
		ID:       row.ID,
		NotAfter: pgconv.TimestampFrom(now.Add(devicePollSlack - deviceLoginInterval)),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "touch login grant")
	}
	if touched == 0 {
		return pollStatus(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_SLOW_DOWN,
			deviceLoginInterval+deviceLoginSlowDown), nil
	}
	return pollStatus(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING, deviceLoginInterval), nil
}

func (h *Handler) redeemGrant(
	ctx context.Context, row db.LoginGrant,
) (*connect.Response[authv1.PollDeviceLoginResponse], error) {
	consumed, err := h.q.ConsumeLoginGrant(ctx, row.ID)
	if err != nil {
		return nil, h.internal(ctx, err, "consume login grant")
	}
	if consumed == 0 {
		return pollStatus(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED, 0), nil
	}

	user, err := h.q.GetUserByID(ctx, row.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pollStatus(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED, 0), nil
		}
		return nil, h.internal(ctx, err, "get user")
	}

	chainID, err := newChainID()
	if err != nil {
		return nil, h.internal(ctx, err, "generate chain id")
	}

	tokens, err := h.mintSession(ctx, user, chainID, true)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&authv1.PollDeviceLoginResponse{
		Status:       authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_APPROVED,
		AccessToken:  tokens.access,
		RefreshToken: tokens.refresh,
		ExpiresIn:    tokens.expiresIn,
	}), nil
}

func (h *Handler) ApproveDeviceLogin(
	ctx context.Context, req *connect.Request[authv1.ApproveDeviceLoginRequest],
) (*connect.Response[authv1.ApproveDeviceLoginResponse], error) {
	userID, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}

	key := approveThrottleKey(userID)
	if wait := h.throttle.Retry(key); wait > 0 {
		return nil, errTooManyAttempts(wait)
	}

	code, ok := normalizeUserCode(req.Msg.GetUserCode())
	if !ok {
		h.throttle.Failed(key)
		return nil, errInvalidUserCode()
	}

	row, err := h.q.ApproveLoginGrant(ctx, db.ApproveLoginGrantParams{
		UserID:       userID,
		DecidedAt:    pgconv.TimestampFrom(h.now()),
		UserCodeHash: hashToken(code),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		row, err = h.q.GetLoginGrantByUserCode(ctx, hashToken(code))
		if errors.Is(err, pgx.ErrNoRows) || err == nil && !(row.ApprovedAt.Valid && row.UserID == userID) {
			h.throttle.Failed(key)
			return nil, errInvalidUserCode()
		}
	}
	if err != nil {
		return nil, h.internal(ctx, err, "approve login grant")
	}

	return connect.NewResponse(&authv1.ApproveDeviceLoginResponse{Kind: kindOf(row.Kind)}), nil
}

func (h *Handler) DenyDeviceLogin(
	ctx context.Context, req *connect.Request[authv1.DenyDeviceLoginRequest],
) (*connect.Response[authv1.DenyDeviceLoginResponse], error) {
	userID, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}

	key := approveThrottleKey(userID)
	if wait := h.throttle.Retry(key); wait > 0 {
		return nil, errTooManyAttempts(wait)
	}

	code, ok := normalizeUserCode(req.Msg.GetUserCode())
	if !ok {
		h.throttle.Failed(key)
		return nil, errInvalidUserCode()
	}

	denied, err := h.q.DenyLoginGrant(ctx, db.DenyLoginGrantParams{
		UserID:       userID,
		DecidedAt:    pgconv.TimestampFrom(h.now()),
		UserCodeHash: hashToken(code),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "deny login grant")
	}
	if denied == 0 {
		row, err := h.q.GetLoginGrantByUserCode(ctx, hashToken(code))
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, h.internal(ctx, err, "get login grant")
		}
		if err != nil || !(row.DeniedAt.Valid && row.UserID == userID) {
			h.throttle.Failed(key)
			return nil, errInvalidUserCode()
		}
	}

	return connect.NewResponse(&authv1.DenyDeviceLoginResponse{}), nil
}

func pollStatus(
	status authv1.DeviceLoginStatus, interval time.Duration,
) *connect.Response[authv1.PollDeviceLoginResponse] {
	return connect.NewResponse(&authv1.PollDeviceLoginResponse{
		Status:          status,
		IntervalSeconds: int32(interval.Seconds()),
	})
}

func kindOf(name string) authv1.DeviceLoginKind {
	for kind, n := range deviceLoginKinds {
		if n == name {
			return kind
		}
	}
	return authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_UNSPECIFIED
}

func newUserCode() (string, error) {
	limit := big.NewInt(int64(len(userCodeAlphabet)))
	var sb strings.Builder
	for range userCodeLength {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		sb.WriteByte(userCodeAlphabet[n.Int64()])
	}
	return sb.String(), nil
}

func formatUserCode(code string) string {
	half := len(code) / 2
	return code[:half] + "-" + code[half:]
}

func normalizeUserCode(raw string) (string, bool) {
	var sb strings.Builder
	for _, r := range strings.ToUpper(raw) {
		switch {
		case r == '-' || r == ' ':
			continue
		case !strings.ContainsRune(userCodeAlphabet, r):
			return "", false
		}
		sb.WriteRune(r)
		if sb.Len() > userCodeLength {
			return "", false
		}
	}
	if sb.Len() != userCodeLength {
		return "", false
	}
	return sb.String(), true
}

func approveThrottleKey(userID pgtype.UUID) string {
	return "device-login:" + pgconv.UUIDString(userID)
}

func errInvalidUserCode() error {
	return connect.NewError(connect.CodeNotFound, errors.New("invalid or expired code"))
}
