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

func (h *Handler) ListAccounts(
	ctx context.Context, req *connect.Request[financev1.ListAccountsRequest],
) (*connect.Response[financev1.ListAccountsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}

	rows, err := h.q.ListVisibleAccounts(ctx, db.ListVisibleAccountsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list accounts")
	}

	out := &financev1.ListAccountsResponse{}
	for _, r := range rows {
		account := toProtoAccount(viewFromList(r))
		if r.Visibility == visibilityPrivate {
			out.PrivateOwn = append(out.PrivateOwn, account)
			continue
		}
		out.Shared = append(out.Shared, account)
	}

	hidden, err := h.q.CountHiddenPrivateAccounts(ctx, db.CountHiddenPrivateAccountsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "count hidden private accounts")
	}
	names, err := h.memberNames(ctx, c)
	if err != nil {
		return nil, err
	}
	for _, row := range hidden {
		id := pgconv.UUIDString(row.OwnerMemberID)
		out.Hidden = append(out.Hidden, &financev1.HiddenPrivateSummary{
			MemberId:          id,
			MemberDisplayName: names[id],
			AccountCount:      row.AccountCount,
		})
	}

	balances, err := h.q.SumFamilyBalances(ctx, db.SumFamilyBalancesParams{
		FamilyID: c.familyID, CurrencyCode: hh.currency(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum family balances")
	}
	out.SharedBalance = money(balances.SharedBalanceMinor, hh.currency())
	out.SavingsTotal = money(balances.SavingsMinor, hh.currency())
	return connect.NewResponse(out), nil
}

func (h *Handler) memberNames(ctx context.Context, c caller) (map[string]string, error) {
	rows, err := h.q.ListMembers(ctx, db.ListMembersParams{
		FamilyID: c.familyID, IncludePending: true,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list members")
	}
	names := make(map[string]string, len(rows))
	for _, m := range rows {
		names[pgconv.UUIDString(m.UserID)] = m.DisplayName
	}
	return names, nil
}

func (h *Handler) GetAccount(
	ctx context.Context, req *connect.Request[financev1.GetAccountRequest],
) (*connect.Response[financev1.GetAccountResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("account_id", req.Msg.GetAccountId())
	if err != nil {
		return nil, err
	}
	row, err := h.q.GetVisibleAccount(ctx, db.GetVisibleAccountParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("account")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get account")
	}
	return connect.NewResponse(&financev1.GetAccountResponse{
		Account: toProtoAccount(viewFromGet(row)),
	}), nil
}

func (h *Handler) CreateAccount(
	ctx context.Context, req *connect.Request[financev1.CreateAccountRequest],
) (*connect.Response[financev1.CreateAccountResponse], error) {
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
	if err := checkText("icon", msg.GetIcon(), maxIconRunes); err != nil {
		return nil, err
	}
	if err := checkColorStep(msg.GetColorStep()); err != nil {
		return nil, err
	}
	currency := hh.currency()
	if trimmed(msg.GetCurrencyCode()) != "" {
		currency, err = checkCurrency(msg.GetCurrencyCode())
		if err != nil {
			return nil, err
		}
	}
	if err := checkMoneyCurrency(msg.GetOpeningBalance(), currency); err != nil {
		return nil, err
	}
	opening := moneyMinor(msg.GetOpeningBalance())

	visibility := visibilityFromProto(msg.GetVisibility())
	var owner pgtype.UUID
	excluded := msg.GetExcludedFromFamilyTotal()
	if visibility == visibilityPrivate {
		owner = c.memberID()
		excluded = true
	}

	row, err := h.q.CreateAccount(ctx, db.CreateAccountParams{
		FamilyID: c.familyID, Name: name, Kind: accountKindFromProto(msg.GetKind()),
		Visibility: visibility, OwnerMemberID: owner, CurrencyCode: currency,
		OpeningBalanceMinor: opening, Icon: trimmed(msg.GetIcon()),
		ColorStep: msg.GetColorStep(), ExcludedFromFamilyTotal: excluded,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create account")
	}

	h.publish(ctx, subjectAccountCreated, &financev1.AccountCreatedEvent{
		FamilyId:      c.family,
		AccountId:     pgconv.UUIDString(row.ID),
		Visibility:    visibilityToProto(row.Visibility),
		OwnerMemberId: pgconv.UUIDString(row.OwnerMemberID),
		Kind:          accountKindToProto(row.Kind),
		OccurredAt:    h.timestamp(),
	})
	return connect.NewResponse(&financev1.CreateAccountResponse{
		Account: toProtoAccount(viewFromWrite(row)),
	}), nil
}

func (h *Handler) UpdateAccount(
	ctx context.Context, req *connect.Request[financev1.UpdateAccountRequest],
) (*connect.Response[financev1.UpdateAccountResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("account_id", msg.GetAccountId())
	if err != nil {
		return nil, err
	}
	existing, err := h.q.GetVisibleAccount(ctx, db.GetVisibleAccountParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("account")
		}
		return nil, h.internal(ctx, err, "get account")
	}

	params := db.UpdateAccountParams{ID: id, FamilyID: c.familyID}
	if msg.Name != nil {
		name := trimmed(msg.GetName())
		if name == "" {
			return nil, invalid("name is required")
		}
		if err := checkText("name", name, maxNameRunes); err != nil {
			return nil, err
		}
		params.Name = &name
	}
	if msg.Kind != nil {
		kind := accountKindFromProto(msg.GetKind())
		params.Kind = &kind
	}
	if msg.Icon != nil {
		icon := trimmed(msg.GetIcon())
		if err := checkText("icon", icon, maxIconRunes); err != nil {
			return nil, err
		}
		params.Icon = &icon
	}
	if msg.ColorStep != nil {
		if err := checkColorStep(msg.GetColorStep()); err != nil {
			return nil, err
		}
		step := msg.GetColorStep()
		params.ColorStep = &step
	}
	if msg.ExcludedFromFamilyTotal != nil {
		excluded := msg.GetExcludedFromFamilyTotal()
		params.ExcludedFromFamilyTotal = &excluded
	}
	if msg.OpeningBalance != nil {
		if err := checkMoneyCurrency(msg.GetOpeningBalance(), existing.CurrencyCode); err != nil {
			return nil, err
		}
		opening := moneyMinor(msg.GetOpeningBalance())
		params.OpeningBalanceMinor = &opening
	}

	var row db.Account
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if msg.Visibility != nil {
			visibility := visibilityFromProto(msg.GetVisibility())
			var owner pgtype.UUID
			if visibility == visibilityPrivate {
				owner = c.memberID()
			}
			if _, err := q.SetAccountVisibility(ctx, db.SetAccountVisibilityParams{
				ID: id, FamilyID: c.familyID, Visibility: visibility, OwnerMemberID: owner,
			}); err != nil {
				return err
			}
		}
		updated, err := q.UpdateAccount(ctx, params)
		if err != nil {
			return err
		}
		row = updated
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("account")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update account")
	}

	h.publish(ctx, subjectAccountUpdated, &financev1.AccountUpdatedEvent{
		FamilyId:   c.family,
		AccountId:  pgconv.UUIDString(row.ID),
		Visibility: visibilityToProto(row.Visibility),
		Archived:   row.Archived,
		OccurredAt: h.timestamp(),
	})
	return connect.NewResponse(&financev1.UpdateAccountResponse{
		Account: toProtoAccount(viewFromWrite(row)),
	}), nil
}

func (h *Handler) ArchiveAccount(
	ctx context.Context, req *connect.Request[financev1.ArchiveAccountRequest],
) (*connect.Response[financev1.ArchiveAccountResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("account_id", req.Msg.GetAccountId())
	if err != nil {
		return nil, err
	}
	if _, err := h.q.GetVisibleAccount(ctx, db.GetVisibleAccountParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("account")
		}
		return nil, h.internal(ctx, err, "get account")
	}

	row, err := h.q.SetAccountArchived(ctx, db.SetAccountArchivedParams{
		ID: id, FamilyID: c.familyID, Archived: req.Msg.GetArchived(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("account")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "archive account")
	}
	h.publish(ctx, subjectAccountUpdated, &financev1.AccountUpdatedEvent{
		FamilyId:   c.family,
		AccountId:  pgconv.UUIDString(row.ID),
		Visibility: visibilityToProto(row.Visibility),
		Archived:   row.Archived,
		OccurredAt: h.timestamp(),
	})
	return connect.NewResponse(&financev1.ArchiveAccountResponse{
		Account: toProtoAccount(viewFromWrite(row)),
	}), nil
}

func (h *Handler) DeleteAccount(
	ctx context.Context, req *connect.Request[financev1.DeleteAccountRequest],
) (*connect.Response[financev1.DeleteAccountResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("account_id", req.Msg.GetAccountId())
	if err != nil {
		return nil, err
	}
	if _, err := h.q.GetVisibleAccount(ctx, db.GetVisibleAccountParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("account")
		}
		return nil, h.internal(ctx, err, "get account")
	}

	count, err := h.q.CountAccountTransactions(ctx, db.CountAccountTransactionsParams{
		AccountID: id, FamilyID: c.familyID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "count account transactions")
	}
	if count > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("this account still has transactions; archive it instead"))
	}

	rows, err := h.q.DeleteAccount(ctx, db.DeleteAccountParams{ID: id, FamilyID: c.familyID})
	if err != nil {
		return nil, h.internal(ctx, err, "delete account")
	}
	if rows == 0 {
		return nil, notFound("account")
	}
	return connect.NewResponse(&financev1.DeleteAccountResponse{}), nil
}

func (h *Handler) ReorderAccounts(
	ctx context.Context, req *connect.Request[financev1.ReorderAccountsRequest],
) (*connect.Response[financev1.ReorderAccountsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	ids := req.Msg.GetAccountIdsInOrder()
	if err := checkBatch("account_ids_in_order", ids); err != nil {
		return nil, err
	}
	parsed, err := uuidList("account_ids_in_order", ids)
	if err != nil {
		return nil, err
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		for i, id := range parsed {
			if err := q.ReorderAccount(ctx, db.ReorderAccountParams{
				ID: id, FamilyID: c.familyID, SortOrder: int32(i),
				ViewerMemberID: c.memberID(),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, h.internal(ctx, err, "reorder accounts")
	}
	return connect.NewResponse(&financev1.ReorderAccountsResponse{}), nil
}

func (h *Handler) TransferBetweenAccounts(
	ctx context.Context, req *connect.Request[financev1.TransferBetweenAccountsRequest],
) (*connect.Response[financev1.TransferBetweenAccountsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg

	fromID, err := requireUUID("from_account_id", msg.GetFromAccountId())
	if err != nil {
		return nil, err
	}
	toID, err := requireUUID("to_account_id", msg.GetToAccountId())
	if err != nil {
		return nil, err
	}
	if pgconv.UUIDString(fromID) == pgconv.UUIDString(toID) {
		return nil, invalid("a transfer needs two different accounts")
	}
	amount, err := checkAmount(msg.GetAmount())
	if err != nil {
		return nil, err
	}
	if amount == 0 {
		return nil, invalid("amount must be greater than zero")
	}
	if err := checkText("note", msg.GetNote(), maxNoteRunes); err != nil {
		return nil, err
	}

	source, err := h.visibleAccount(ctx, c, fromID)
	if err != nil {
		return nil, err
	}
	target, err := h.visibleAccount(ctx, c, toID)
	if err != nil {
		return nil, err
	}

	if err := checkMoneyCurrency(msg.GetAmount(), source.CurrencyCode); err != nil {
		return nil, err
	}
	if msg.GetReceivedAmount() != nil {
		if err := checkMoneyCurrency(msg.GetReceivedAmount(), target.CurrencyCode); err != nil {
			return nil, err
		}
	}

	memberID := c.memberID()
	if trimmed(msg.GetMemberId()) != "" {
		memberID, err = requireUUID("member_id", msg.GetMemberId())
		if err != nil {
			return nil, err
		}
	}
	occurred, err := requireDay("occurred_on", msg.GetOccurredOn(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}

	params := db.CreateTransactionParams{
		FamilyID: c.familyID, Type: kindTransfer, AccountID: fromID, CounterAccountID: toID,
		AmountMinor: amount, CurrencyCode: source.CurrencyCode,
		Note: trimmed(msg.GetNote()), OccurredOn: pgDate(occurred),
		MemberID: memberID, CreatedByUserID: c.userID,
	}
	if source.CurrencyCode != target.CurrencyCode {
		received := moneyMinor(msg.GetReceivedAmount())
		if received <= 0 {
			return nil, invalid("received_amount is required when the accounts hold different currencies")
		}
		params.ReceivedAmountMinor = &received
		params.ReceivedCurrencyCode = target.CurrencyCode
	}

	row, err := h.q.CreateTransaction(ctx, params)
	if err != nil {
		return nil, h.internal(ctx, err, "create transfer")
	}

	event := &financev1.TransferCreatedEvent{
		FamilyId:      c.family,
		TransactionId: pgconv.UUIDString(row.ID),
		FromAccountId: pgconv.UUIDString(row.AccountID),
		ToAccountId:   pgconv.UUIDString(row.CounterAccountID),
		Amount:        money(row.AmountMinor, row.CurrencyCode),
		OccurredOn:    pgconv.DateString(row.OccurredOn),
		ActorUserId:   c.user,
		OccurredAt:    h.timestamp(),
	}
	if row.ReceivedAmountMinor != nil {
		event.ReceivedAmount = money(*row.ReceivedAmountMinor, row.ReceivedCurrencyCode)
	}
	h.publish(ctx, subjectTransferCreated, event)

	return connect.NewResponse(&financev1.TransferBetweenAccountsResponse{
		Transaction: toProtoTransaction(transactionView{row: row}),
	}), nil
}

func (h *Handler) visibleAccount(ctx context.Context, c caller, id pgtype.UUID) (db.GetVisibleAccountRow, error) {
	row, err := h.q.GetVisibleAccount(ctx, db.GetVisibleAccountParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetVisibleAccountRow{}, notFound("account")
	}
	if err != nil {
		return db.GetVisibleAccountRow{}, h.internal(ctx, err, "get account")
	}
	return row, nil
}
