package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

func (h *Handler) ListTemplates(
	ctx context.Context, req *connect.Request[financev1.ListTemplatesRequest],
) (*connect.Response[financev1.ListTemplatesResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if owner := trimmed(req.Msg.GetOwnerUserId()); owner != "" && owner != c.user {
		return connect.NewResponse(&financev1.ListTemplatesResponse{}), nil
	}
	rows, err := h.q.ListTemplates(ctx, db.ListTemplatesParams{
		FamilyID: c.familyID, OwnerUserID: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list templates")
	}
	out := make([]*financev1.QuickTemplate, 0, len(rows))
	for _, t := range rows {
		out = append(out, toProtoTemplate(t))
	}
	return connect.NewResponse(&financev1.ListTemplatesResponse{Templates: out}), nil
}

func (h *Handler) CreateTemplate(
	ctx context.Context, req *connect.Request[financev1.CreateTemplateRequest],
) (*connect.Response[financev1.CreateTemplateResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg

	label := trimmed(msg.GetLabel())
	if label == "" {
		return nil, invalid("label is required")
	}
	if err := checkText("label", label, maxLabelRunes); err != nil {
		return nil, err
	}
	if err := checkText("icon", msg.GetIcon(), maxIconRunes); err != nil {
		return nil, err
	}
	amount, err := checkAmount(msg.GetAmount())
	if err != nil {
		return nil, err
	}
	accountID, err := requireUUID("account_id", msg.GetAccountId())
	if err != nil {
		return nil, err
	}
	account, err := h.visibleAccount(ctx, c, accountID)
	if err != nil {
		return nil, err
	}
	categoryID, err := optionalUUID("category_id", msg.GetCategoryId())
	if err != nil {
		return nil, err
	}
	if err := h.checkCategoryKind(ctx, c, categoryID, txTypeFromProto(msg.GetType())); err != nil {
		return nil, err
	}
	memberID := c.memberID()
	if trimmed(msg.GetMemberId()) != "" {
		if memberID, err = requireUUID("member_id", msg.GetMemberId()); err != nil {
			return nil, err
		}
	}
	currency := account.CurrencyCode
	if currency == "" {
		currency = hh.currency()
	}
	if err := checkMoneyCurrency(msg.GetAmount(), currency); err != nil {
		return nil, err
	}

	row, err := h.q.CreateTemplate(ctx, db.CreateTemplateParams{
		FamilyID: c.familyID, OwnerUserID: c.userID, Label: label,
		Icon: trimmed(msg.GetIcon()), AmountMinor: amount, CurrencyCode: currency,
		Type: txTypeFromProto(msg.GetType()), CategoryID: categoryID,
		AccountID: accountID, MemberID: memberID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create template")
	}
	return connect.NewResponse(&financev1.CreateTemplateResponse{
		Template: toProtoTemplate(row),
	}), nil
}

func (h *Handler) UpdateTemplate(
	ctx context.Context, req *connect.Request[financev1.UpdateTemplateRequest],
) (*connect.Response[financev1.UpdateTemplateResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("template_id", msg.GetTemplateId())
	if err != nil {
		return nil, err
	}

	params := db.UpdateTemplateParams{ID: id, FamilyID: c.familyID, OwnerUserID: c.userID}
	if msg.Label != nil {
		label := trimmed(msg.GetLabel())
		if label == "" {
			return nil, invalid("label is required")
		}
		if err := checkText("label", label, maxLabelRunes); err != nil {
			return nil, err
		}
		params.Label = &label
	}
	if msg.Icon != nil {
		icon := trimmed(msg.GetIcon())
		if err := checkText("icon", icon, maxIconRunes); err != nil {
			return nil, err
		}
		params.Icon = &icon
	}
	tpl, err := h.q.GetTemplate(ctx, db.GetTemplateParams{
		ID: id, FamilyID: c.familyID, OwnerUserID: c.userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("template")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get template")
	}
	currency := tpl.CurrencyCode
	if msg.AccountId != nil {
		accountID, err := requireUUID("account_id", msg.GetAccountId())
		if err != nil {
			return nil, err
		}
		account, err := h.visibleAccount(ctx, c, accountID)
		if err != nil {
			return nil, err
		}
		params.AccountID = accountID
		if account.CurrencyCode != currency {
			if msg.Amount == nil {
				return nil, invalid(
					"this account is in %s: send the amount in %s as well",
					account.CurrencyCode, account.CurrencyCode)
			}
			currency = account.CurrencyCode
			params.CurrencyCode = &currency
		}
	}
	if msg.Amount != nil {
		amount, err := checkAmount(msg.GetAmount())
		if err != nil {
			return nil, err
		}
		if err := checkMoneyCurrency(msg.GetAmount(), currency); err != nil {
			return nil, err
		}
		if amount <= 0 {
			return nil, invalid("amount must be greater than zero")
		}
		params.AmountMinor = &amount
	}
	if msg.CategoryId != nil {
		if params.CategoryID, err = optionalUUID("category_id", msg.GetCategoryId()); err != nil {
			return nil, err
		}
		if err := h.checkCategoryKind(ctx, c, params.CategoryID, tpl.Type); err != nil {
			return nil, err
		}
	}
	if msg.MemberId != nil {
		if params.MemberID, err = requireUUID("member_id", msg.GetMemberId()); err != nil {
			return nil, err
		}
	}

	row, err := h.q.UpdateTemplate(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("template")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update template")
	}
	return connect.NewResponse(&financev1.UpdateTemplateResponse{
		Template: toProtoTemplate(row),
	}), nil
}

func (h *Handler) DeleteTemplate(
	ctx context.Context, req *connect.Request[financev1.DeleteTemplateRequest],
) (*connect.Response[financev1.DeleteTemplateResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("template_id", req.Msg.GetTemplateId())
	if err != nil {
		return nil, err
	}
	rows, err := h.q.DeleteTemplate(ctx, db.DeleteTemplateParams{
		ID: id, FamilyID: c.familyID, OwnerUserID: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "delete template")
	}
	if rows == 0 {
		return nil, notFound("template")
	}
	return connect.NewResponse(&financev1.DeleteTemplateResponse{}), nil
}

func (h *Handler) ReorderTemplates(
	ctx context.Context, req *connect.Request[financev1.ReorderTemplatesRequest],
) (*connect.Response[financev1.ReorderTemplatesResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := checkBatch("template_ids_in_order", req.Msg.GetTemplateIdsInOrder()); err != nil {
		return nil, err
	}
	ids, err := uuidList("template_ids_in_order", req.Msg.GetTemplateIdsInOrder())
	if err != nil {
		return nil, err
	}
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		for i, id := range ids {
			if err := q.ReorderTemplate(ctx, db.ReorderTemplateParams{
				ID: id, FamilyID: c.familyID, OwnerUserID: c.userID, SortOrder: int32(i),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, h.internal(ctx, err, "reorder templates")
	}
	return connect.NewResponse(&financev1.ReorderTemplatesResponse{}), nil
}

func (h *Handler) LogTemplate(
	ctx context.Context, req *connect.Request[financev1.LogTemplateRequest],
) (*connect.Response[financev1.LogTemplateResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("template_id", msg.GetTemplateId())
	if err != nil {
		return nil, err
	}

	tpl, err := h.q.GetTemplate(ctx, db.GetTemplateParams{
		ID: id, FamilyID: c.familyID, OwnerUserID: c.userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("template")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get template")
	}

	amount := tpl.AmountMinor
	if msg.GetAmountOverride() != nil {
		if amount, err = checkAmount(msg.GetAmountOverride()); err != nil {
			return nil, err
		}
		if err := checkMoneyCurrency(msg.GetAmountOverride(), tpl.CurrencyCode); err != nil {
			return nil, err
		}
	}
	if amount <= 0 {
		return nil, invalid("amount must be greater than zero")
	}
	occurred, err := requireDay("occurred_on", msg.GetOccurredOn(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}
	if _, err := h.visibleAccount(ctx, c, tpl.AccountID); err != nil {
		return nil, err
	}

	before, err := h.affectedBudgets(ctx, c, hh, tpl.CategoryID, occurred)
	if err != nil {
		return nil, err
	}

	var row db.Transaction
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		created, err := q.CreateTransaction(ctx, db.CreateTransactionParams{
			FamilyID: c.familyID, Type: tpl.Type, AccountID: tpl.AccountID,
			CategoryID: tpl.CategoryID, AmountMinor: amount, CurrencyCode: tpl.CurrencyCode,
			OccurredOn: pgDate(occurred), MemberID: tpl.MemberID, CreatedByUserID: c.userID,
			TemplateID: tpl.ID,
		})
		if err != nil {
			return err
		}
		row = created
		_, err = q.RecordTemplateUse(ctx, db.RecordTemplateUseParams{
			ID: tpl.ID, FamilyID: c.familyID, OwnerUserID: c.userID,
		})
		return err
	})
	if err != nil {
		return nil, h.internal(ctx, err, "log template")
	}

	view, err := h.attachGroup(ctx, c, row)
	if err != nil {
		return nil, err
	}
	after, err := h.affectedBudgets(ctx, c, hh, tpl.CategoryID, occurred)
	if err != nil {
		return nil, err
	}

	h.announceTransactionCreated(ctx, c, view)
	h.publish(ctx, subjectTemplateUsed, &financev1.TemplateUsedEvent{
		FamilyId:      c.family,
		TemplateId:    pgconv.UUIDString(tpl.ID),
		OwnerUserId:   c.user,
		TransactionId: pgconv.UUIDString(row.ID),
		OccurredAt:    h.timestamp(),
	})
	h.announceBudgetChanges(ctx, c, before, after, pgconv.UUIDString(row.ID))

	return connect.NewResponse(&financev1.LogTemplateResponse{
		Transaction:     toProtoTransaction(view),
		AffectedBudgets: after,
	}), nil
}
