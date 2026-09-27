package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

func investmentKindToProto(s string) financev1.InvestmentKind {
	switch s {
	case "deposit":
		return financev1.InvestmentKind_INVESTMENT_KIND_DEPOSIT
	case "stocks":
		return financev1.InvestmentKind_INVESTMENT_KIND_STOCKS
	case "bonds":
		return financev1.InvestmentKind_INVESTMENT_KIND_BONDS
	case "crypto":
		return financev1.InvestmentKind_INVESTMENT_KIND_CRYPTO
	case "real_estate":
		return financev1.InvestmentKind_INVESTMENT_KIND_REAL_ESTATE
	case "other":
		return financev1.InvestmentKind_INVESTMENT_KIND_OTHER
	}
	return financev1.InvestmentKind_INVESTMENT_KIND_UNSPECIFIED
}

func investmentKindFromProto(k financev1.InvestmentKind) string {
	switch k {
	case financev1.InvestmentKind_INVESTMENT_KIND_DEPOSIT:
		return "deposit"
	case financev1.InvestmentKind_INVESTMENT_KIND_STOCKS:
		return "stocks"
	case financev1.InvestmentKind_INVESTMENT_KIND_BONDS:
		return "bonds"
	case financev1.InvestmentKind_INVESTMENT_KIND_CRYPTO:
		return "crypto"
	case financev1.InvestmentKind_INVESTMENT_KIND_REAL_ESTATE:
		return "real_estate"
	case financev1.InvestmentKind_INVESTMENT_KIND_OTHER:
		return "other"
	}
	return ""
}

var investmentIcons = map[string]string{
	"deposit":     "piggy-bank",
	"stocks":      "trend-up",
	"bonds":       "chart-bar",
	"crypto":      "currency-btc",
	"real_estate": "house-line",
	"other":       "chart-donut",
}

func profitBps(invested, value int64) int32 {
	if invested <= 0 {
		return 0
	}
	return int32((value - invested) * 10000 / invested)
}

func toProtoInvestment(r db.ListInvestmentsRow) *financev1.Investment {
	return &financev1.Investment{
		Id:             pgconv.UUIDString(r.ID),
		FamilyId:       pgconv.UUIDString(r.FamilyID),
		Name:           r.Name,
		Kind:           investmentKindToProto(r.Kind),
		CategoryId:     pgconv.UUIDString(r.CategoryID),
		GroupId:        pgconv.UUIDString(r.GroupID),
		Invested:       money(r.InvestedMinor, r.CurrencyCode),
		CurrentValue:   money(r.CurrentValueMinor, r.CurrencyCode),
		Profit:         money(r.CurrentValueMinor-r.InvestedMinor, r.CurrencyCode),
		ProfitBps:      profitBps(r.InvestedMinor, r.CurrentValueMinor),
		ValueUpdatedOn: pgconv.DateString(r.ValueUpdatedOn),
		Archived:       r.Archived,
		SortOrder:      r.SortOrder,
		CreatedAt:      pgconv.Timestamp(r.CreatedAt),
		UpdatedAt:      pgconv.Timestamp(r.UpdatedAt),
	}
}

func (h *Handler) investmentView(ctx context.Context, c caller, id pgtype.UUID) (*financev1.Investment, error) {
	rows, err := h.q.ListInvestments(ctx, db.ListInvestmentsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(), IncludeArchived: true,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list investments")
	}
	for _, r := range rows {
		if r.ID == id {
			return toProtoInvestment(r), nil
		}
	}
	return nil, notFound("investment")
}

func (h *Handler) ListInvestments(
	ctx context.Context, req *connect.Request[financev1.ListInvestmentsRequest],
) (*connect.Response[financev1.ListInvestmentsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListInvestments(ctx, db.ListInvestmentsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list investments")
	}
	out := make([]*financev1.Investment, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProtoInvestment(r))
	}
	return connect.NewResponse(&financev1.ListInvestmentsResponse{Investments: out}), nil
}

func (h *Handler) CreateInvestment(
	ctx context.Context, req *connect.Request[financev1.CreateInvestmentRequest],
) (*connect.Response[financev1.CreateInvestmentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg

	name := trimmed(msg.GetName())
	if name == "" {
		return nil, invalid("name is required")
	}
	if err := checkText("name", name, maxNameRunes); err != nil {
		return nil, err
	}
	kind := investmentKindFromProto(msg.GetKind())
	if kind == "" {
		kind = "other"
	}
	currency := hh.currency()
	if trimmed(msg.GetCurrencyCode()) != "" {
		if currency, err = checkCurrency(msg.GetCurrencyCode()); err != nil {
			return nil, err
		}
	}

	var created db.Investment
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		category, err := createOwnedCategory(ctx, q, c.familyID, roleInvestments, name, investmentIcons[kind])
		if err != nil {
			return err
		}
		created, err = q.CreateInvestment(ctx, db.CreateInvestmentParams{
			FamilyID: c.familyID, Name: name, Kind: kind,
			CurrencyCode: currency, CategoryID: category.ID,
		})
		return err
	})
	if pgErrorCode(err) == pgUniqueViolation {
		return nil, connect.NewError(connect.CodeAlreadyExists,
			errors.New("an investment with this name already exists"))
	}
	if err != nil {
		return nil, h.internal(ctx, err, "create investment")
	}

	view, err := h.investmentView(ctx, c, created.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.CreateInvestmentResponse{Investment: view}), nil
}

func (h *Handler) getInvestment(ctx context.Context, c caller, raw string) (db.Investment, error) {
	id, err := requireUUID("investment_id", raw)
	if err != nil {
		return db.Investment{}, err
	}
	row, err := h.q.GetInvestment(ctx, db.GetInvestmentParams{ID: id, FamilyID: c.familyID})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Investment{}, notFound("investment")
	}
	if err != nil {
		return db.Investment{}, h.internal(ctx, err, "get investment")
	}
	return row, nil
}

func (h *Handler) UpdateInvestment(
	ctx context.Context, req *connect.Request[financev1.UpdateInvestmentRequest],
) (*connect.Response[financev1.UpdateInvestmentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	inv, err := h.getInvestment(ctx, c, msg.GetInvestmentId())
	if err != nil {
		return nil, err
	}

	params := db.UpdateInvestmentParams{ID: inv.ID, FamilyID: c.familyID}
	if msg.Name != nil {
		name := trimmed(msg.GetName())
		if name == "" {
			return nil, invalid("name must not be empty")
		}
		if err := checkText("name", name, maxNameRunes); err != nil {
			return nil, err
		}
		params.Name = &name
	}
	if kind := investmentKindFromProto(msg.GetKind()); kind != "" {
		params.Kind = &kind
	}
	if msg.Archived != nil {
		archived := msg.GetArchived()
		params.Archived = &archived
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if _, err := q.UpdateInvestment(ctx, params); err != nil {
			return err
		}
		if params.Name != nil || params.Kind != nil || params.Archived != nil {
			catParams := db.UpdateCategoryParams{ID: inv.CategoryID, FamilyID: c.familyID,
				Name: params.Name, Archived: params.Archived}
			if params.Kind != nil {
				icon := investmentIcons[*params.Kind]
				catParams.Icon = &icon
			}
			if _, err := q.UpdateCategory(ctx, catParams); err != nil {
				return err
			}
		}
		return nil
	})
	if pgErrorCode(err) == pgUniqueViolation {
		return nil, connect.NewError(connect.CodeAlreadyExists,
			errors.New("an investment or category with this name already exists"))
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update investment")
	}

	view, err := h.investmentView(ctx, c, inv.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.UpdateInvestmentResponse{Investment: view}), nil
}

func (h *Handler) SetInvestmentValue(
	ctx context.Context, req *connect.Request[financev1.SetInvestmentValueRequest],
) (*connect.Response[financev1.SetInvestmentValueResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	inv, err := h.getInvestment(ctx, c, msg.GetInvestmentId())
	if err != nil {
		return nil, err
	}
	value, err := checkAmount(msg.GetValue())
	if err != nil {
		return nil, err
	}
	if err := checkMoneyCurrency(msg.GetValue(), inv.CurrencyCode); err != nil {
		return nil, err
	}
	valuedOn, err := requireDay("valued_on", msg.GetValuedOn(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}

	if _, err := h.q.SetInvestmentValue(ctx, db.SetInvestmentValueParams{
		ID: inv.ID, FamilyID: c.familyID, CurrentValueMinor: value, ValueUpdatedOn: pgDate(valuedOn),
	}); err != nil {
		return nil, h.internal(ctx, err, "set investment value")
	}

	view, err := h.investmentView(ctx, c, inv.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.SetInvestmentValueResponse{Investment: view}), nil
}

func (h *Handler) DeleteInvestment(
	ctx context.Context, req *connect.Request[financev1.DeleteInvestmentRequest],
) (*connect.Response[financev1.DeleteInvestmentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := h.getInvestment(ctx, c, req.Msg.GetInvestmentId())
	if err != nil {
		return nil, err
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		count, err := q.CountCategoryTransactions(ctx, db.CountCategoryTransactionsParams{
			CategoryID: inv.CategoryID, FamilyID: c.familyID,
		})
		if err != nil {
			return err
		}
		if count > 0 {
			return connect.NewError(connect.CodeFailedPrecondition,
				errors.New("this investment has contributions; archive it instead"))
		}
		if _, err := q.DeleteInvestment(ctx, db.DeleteInvestmentParams{ID: inv.ID, FamilyID: c.familyID}); err != nil {
			return err
		}
		_, err = q.DeleteCategory(ctx, db.DeleteCategoryParams{ID: inv.CategoryID, FamilyID: c.familyID})
		return err
	})
	var cerr *connect.Error
	if errors.As(err, &cerr) {
		return nil, cerr
	}
	if err != nil {
		return nil, h.internal(ctx, err, "delete investment")
	}
	return connect.NewResponse(&financev1.DeleteInvestmentResponse{}), nil
}
