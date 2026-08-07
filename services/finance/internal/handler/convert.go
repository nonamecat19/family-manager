package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// Stored enum values. The CHECK constraints in 000001_init.up.sql are the other half of this
// contract — change one and the other rejects the write.
const (
	typeExpense  = "expense"
	typeIncome   = "income"
	typeTransfer = "transfer"

	accountCash    = "cash"
	accountCard    = "card"
	accountBank    = "bank"
	accountSavings = "savings"
	accountDebt    = "debt"
)

// EventBus is the slice of libs/go/events this service uses.
type EventBus interface {
	Publish(ctx context.Context, subject events.Subject, msg proto.Message) error
}

type noopBus struct{}

func (noopBus) Publish(context.Context, events.Subject, proto.Message) error { return nil }

// publish is fire-and-forget: a ledger write that succeeded must not be reported as failed
// because the broker hiccuped.
func (h *Handler) publish(ctx context.Context, subject events.Subject, msg proto.Message) {
	if err := h.bus.Publish(ctx, subject, msg); err != nil {
		h.log.WarnContext(ctx, "publish failed", slog.String("subject", string(subject)),
			slog.String("error", err.Error()))
	}
}

func (h *Handler) timestamp() *timestamppb.Timestamp {
	return timestamppb.New(h.now())
}

func trimmed(s string) string { return strings.TrimSpace(s) }

/* --------------------------------------------------------------------- enums */

func txTypeToStored(t financev1.TransactionType) (string, bool) {
	switch t {
	case financev1.TransactionType_TRANSACTION_TYPE_EXPENSE:
		return typeExpense, true
	case financev1.TransactionType_TRANSACTION_TYPE_INCOME:
		return typeIncome, true
	case financev1.TransactionType_TRANSACTION_TYPE_TRANSFER:
		return typeTransfer, true
	default:
		return "", false
	}
}

func txTypeToProto(s string) financev1.TransactionType {
	switch s {
	case typeExpense:
		return financev1.TransactionType_TRANSACTION_TYPE_EXPENSE
	case typeIncome:
		return financev1.TransactionType_TRANSACTION_TYPE_INCOME
	case typeTransfer:
		return financev1.TransactionType_TRANSACTION_TYPE_TRANSFER
	default:
		return financev1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

func accountTypeToStored(t financev1.AccountType) string {
	switch t {
	case financev1.AccountType_ACCOUNT_TYPE_CARD:
		return accountCard
	case financev1.AccountType_ACCOUNT_TYPE_BANK:
		return accountBank
	case financev1.AccountType_ACCOUNT_TYPE_SAVINGS:
		return accountSavings
	case financev1.AccountType_ACCOUNT_TYPE_DEBT:
		return accountDebt
	default:
		// Unspecified means cash: the default account a user creates is a wallet.
		return accountCash
	}
}

func accountTypeToProto(s string) financev1.AccountType {
	switch s {
	case accountCash:
		return financev1.AccountType_ACCOUNT_TYPE_CASH
	case accountCard:
		return financev1.AccountType_ACCOUNT_TYPE_CARD
	case accountBank:
		return financev1.AccountType_ACCOUNT_TYPE_BANK
	case accountSavings:
		return financev1.AccountType_ACCOUNT_TYPE_SAVINGS
	case accountDebt:
		return financev1.AccountType_ACCOUNT_TYPE_DEBT
	default:
		return financev1.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	}
}

/* -------------------------------------------------------------------- money */

func money(amountMinor int64, currency string) *financev1.Money {
	return &financev1.Money{AmountMinor: amountMinor, CurrencyCode: currency}
}

/* --------------------------------------------------------------- row mapping */

func toProtoAccount(a db.Account, balanceMinor int64) *financev1.Account {
	return &financev1.Account{
		Id:             pgconv.UUIDString(a.ID),
		FamilyId:       pgconv.UUIDString(a.FamilyID),
		Name:           a.Name,
		Type:           accountTypeToProto(a.Type),
		CurrencyCode:   a.CurrencyCode,
		Balance:        money(balanceMinor, a.CurrencyCode),
		OpeningBalance: money(a.OpeningBalanceMinor, a.CurrencyCode),
		Color:          a.Color,
		Icon:           a.Icon,
		Archived:       a.Archived,
		SortOrder:      a.SortOrder,
		CreatedAt:      pgconv.Timestamp(a.CreatedAt),
		UpdatedAt:      pgconv.Timestamp(a.UpdatedAt),
	}
}

func accountFromBalanceRow(r db.ListAccountsWithBalanceRow) *financev1.Account {
	return toProtoAccount(db.Account{
		ID:                  r.ID,
		FamilyID:            r.FamilyID,
		Name:                r.Name,
		Type:                r.Type,
		CurrencyCode:        r.CurrencyCode,
		OpeningBalanceMinor: r.OpeningBalanceMinor,
		Color:               r.Color,
		Icon:                r.Icon,
		Archived:            r.Archived,
		SortOrder:           r.SortOrder,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}, r.BalanceMinor)
}

func toProtoCategory(c db.Category) *financev1.Category {
	return &financev1.Category{
		Id:        pgconv.UUIDString(c.ID),
		FamilyId:  pgconv.UUIDString(c.FamilyID),
		Name:      c.Name,
		Kind:      txTypeToProto(c.Kind),
		Color:     c.Color,
		Icon:      c.Icon,
		ParentId:  pgconv.UUIDString(c.ParentID),
		Archived:  c.Archived,
		SortOrder: c.SortOrder,
		CreatedAt: pgconv.Timestamp(c.CreatedAt),
		UpdatedAt: pgconv.Timestamp(c.UpdatedAt),
	}
}

func toProtoTransaction(t db.Transaction) *financev1.Transaction {
	return &financev1.Transaction{
		Id:               pgconv.UUIDString(t.ID),
		FamilyId:         pgconv.UUIDString(t.FamilyID),
		AccountId:        pgconv.UUIDString(t.AccountID),
		CounterAccountId: pgconv.UUIDString(t.CounterAccountID),
		CategoryId:       pgconv.UUIDString(t.CategoryID),
		Type:             txTypeToProto(t.Type),
		Amount:           money(t.AmountMinor, t.CurrencyCode),
		Note:             t.Note,
		OccurredOn:       pgconv.DateString(t.OccurredOn),
		CreatedByUserId:  pgconv.UUIDString(t.CreatedByUserID),
		CreatedAt:        pgconv.Timestamp(t.CreatedAt),
		UpdatedAt:        pgconv.Timestamp(t.UpdatedAt),
	}
}

/* ------------------------------------------------------------------- cursors */

// encodeCursor packs the keyset position. It is opaque to clients on purpose: the moment a
// client parses it, the sort order can never change.
func encodeCursor(occurredOn, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(occurredOn + "|" + id))
}

func decodeCursor(token string) (occurredOn, id string, err error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", fmt.Errorf("bad page token")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("bad page token")
	}
	return parts[0], parts[1], nil
}

// slogError is the one-liner every best-effort path uses to log why it gave up.
func slogError(err error) slog.Attr {
	return slog.String("error", err.Error())
}
