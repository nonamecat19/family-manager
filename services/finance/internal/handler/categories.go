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

// ListCategoryTree is the whole categories screen and the add sheet's picker in one call: the
// groups, their categories, and the budget status drawn under each group heading. Three
// round trips would let the app render a group whose budget bar came from a different moment.
func (h *Handler) ListCategoryTree(
	ctx context.Context, req *connect.Request[financev1.ListCategoryTreeRequest],
) (*connect.Response[financev1.ListCategoryTreeResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	asOf, err := requireDay("as_of", req.Msg.GetAsOf(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}

	groups, err := h.q.ListCategoryGroups(ctx, db.ListCategoryGroupsParams{
		FamilyID: c.familyID, Kind: kindFilter(req.Msg.GetKind()),
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list category groups")
	}
	categories, err := h.q.ListCategories(ctx, db.ListCategoriesParams{
		FamilyID: c.familyID, Kind: kindFilter(req.Msg.GetKind()),
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list categories")
	}
	byGroup := map[string][]db.Category{}
	for _, cat := range categories {
		id := pgconv.UUIDString(cat.GroupID)
		byGroup[id] = append(byGroup[id], cat)
	}

	budgets, err := h.q.ListBudgets(ctx, db.ListBudgetsParams{
		FamilyID: c.familyID, TargetKind: strPtr(targetGroup),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list budgets")
	}
	budgetByGroup := map[string]db.Budget{}
	for _, b := range budgets {
		budgetByGroup[pgconv.UUIDString(b.GroupID)] = b
	}

	nodes := make([]*financev1.GroupNode, 0, len(groups))
	for _, g := range groups {
		id := pgconv.UUIDString(g.ID)
		node := &financev1.GroupNode{
			Group:         toProtoGroup(g),
			CategoryCount: int32(len(byGroup[id])),
		}
		for _, cat := range byGroup[id] {
			node.Categories = append(node.Categories, toProtoCategory(cat))
		}
		if b, ok := budgetByGroup[id]; ok {
			status, err := h.budgetStatus(ctx, c, hh, b, asOf)
			if err != nil {
				return nil, err
			}
			node.Budget = status
		}
		nodes = append(nodes, node)
	}
	return connect.NewResponse(&financev1.ListCategoryTreeResponse{Groups: nodes}), nil
}

func (h *Handler) CreateCategoryGroup(
	ctx context.Context, req *connect.Request[financev1.CreateCategoryGroupRequest],
) (*connect.Response[financev1.CreateCategoryGroupResponse], error) {
	c, err := h.caller(ctx)
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

	row, err := h.q.CreateCategoryGroup(ctx, db.CreateCategoryGroupParams{
		FamilyID: c.familyID, Name: name, Kind: taxonomyKindFromProto(msg.GetKind()),
		Icon: trimmed(msg.GetIcon()), ColorStep: msg.GetColorStep(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create category group")
	}
	return connect.NewResponse(&financev1.CreateCategoryGroupResponse{
		Group: toProtoGroup(row),
	}), nil
}

func (h *Handler) UpdateCategoryGroup(
	ctx context.Context, req *connect.Request[financev1.UpdateCategoryGroupRequest],
) (*connect.Response[financev1.UpdateCategoryGroupResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("group_id", msg.GetGroupId())
	if err != nil {
		return nil, err
	}

	params := db.UpdateCategoryGroupParams{ID: id, FamilyID: c.familyID}
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
	if msg.Archived != nil {
		archived := msg.GetArchived()
		params.Archived = &archived
	}

	row, err := h.q.UpdateCategoryGroup(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("category group")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update category group")
	}
	return connect.NewResponse(&financev1.UpdateCategoryGroupResponse{
		Group: toProtoGroup(row),
	}), nil
}

// DeleteCategoryGroup refuses to make orphans: a group holding categories that hold
// transactions cannot silently vanish, so the request has to name where its categories go.
func (h *Handler) DeleteCategoryGroup(
	ctx context.Context, req *connect.Request[financev1.DeleteCategoryGroupRequest],
) (*connect.Response[financev1.DeleteCategoryGroupResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("group_id", req.Msg.GetGroupId())
	if err != nil {
		return nil, err
	}
	target, err := optionalUUID("reassign_to_group_id", req.Msg.GetReassignToGroupId())
	if err != nil {
		return nil, err
	}
	if target.Valid && pgconv.UUIDString(target) == pgconv.UUIDString(id) {
		return nil, invalid("reassign_to_group_id must be a different group")
	}

	var moved int64
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if _, err := q.GetCategoryGroup(ctx, db.GetCategoryGroupParams{
			ID: id, FamilyID: c.familyID,
		}); err != nil {
			return err
		}
		count, err := q.CountCategoriesInGroup(ctx, db.CountCategoriesInGroupParams{
			GroupID: id, FamilyID: c.familyID,
		})
		if err != nil {
			return err
		}
		if count > 0 {
			if !target.Valid {
				return connect.NewError(connect.CodeFailedPrecondition,
					errors.New("this group still holds categories; choose a group to move them to"))
			}
			if _, err := q.GetCategoryGroup(ctx, db.GetCategoryGroupParams{
				ID: target, FamilyID: c.familyID,
			}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return notFound("reassignment group")
				}
				return err
			}
			moved, err = q.MoveCategoriesToGroup(ctx, db.MoveCategoriesToGroupParams{
				GroupID: id, FamilyID: c.familyID, GroupID_2: target,
			})
			if err != nil {
				return err
			}
		}
		rows, err := q.DeleteCategoryGroup(ctx, db.DeleteCategoryGroupParams{
			ID: id, FamilyID: c.familyID,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("category group")
	}
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			return nil, connectErr
		}
		return nil, h.internal(ctx, err, "delete category group")
	}
	return connect.NewResponse(&financev1.DeleteCategoryGroupResponse{
		MovedCategories: int32(moved),
	}), nil
}

func (h *Handler) ReorderCategoryGroups(
	ctx context.Context, req *connect.Request[financev1.ReorderCategoryGroupsRequest],
) (*connect.Response[financev1.ReorderCategoryGroupsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := checkBatch("group_ids_in_order", req.Msg.GetGroupIdsInOrder()); err != nil {
		return nil, err
	}
	ids, err := uuidList("group_ids_in_order", req.Msg.GetGroupIdsInOrder())
	if err != nil {
		return nil, err
	}
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		for i, id := range ids {
			if err := q.ReorderCategoryGroup(ctx, db.ReorderCategoryGroupParams{
				ID: id, FamilyID: c.familyID, SortOrder: int32(i),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, h.internal(ctx, err, "reorder category groups")
	}
	return connect.NewResponse(&financev1.ReorderCategoryGroupsResponse{}), nil
}

func (h *Handler) CreateCategory(
	ctx context.Context, req *connect.Request[financev1.CreateCategoryRequest],
) (*connect.Response[financev1.CreateCategoryResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	groupID, err := requireUUID("group_id", msg.GetGroupId())
	if err != nil {
		return nil, err
	}
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

	group, err := h.q.GetCategoryGroup(ctx, db.GetCategoryGroupParams{
		ID: groupID, FamilyID: c.familyID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("category group")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get category group")
	}
	// A category inherits its group's kind unless the request states one: an income category
	// inside an expense group would appear on neither tab.
	kind := group.Kind
	if msg.GetKind() != financev1.TransactionKind_TRANSACTION_KIND_UNSPECIFIED {
		kind = taxonomyKindFromProto(msg.GetKind())
		if kind != group.Kind {
			return nil, invalid("a category's kind must match its group's")
		}
	}

	row, err := h.q.CreateCategory(ctx, db.CreateCategoryParams{
		FamilyID: c.familyID, GroupID: groupID, Name: name, Kind: kind,
		Icon: trimmed(msg.GetIcon()),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create category")
	}
	return connect.NewResponse(&financev1.CreateCategoryResponse{
		Category: toProtoCategory(row),
	}), nil
}

func (h *Handler) UpdateCategory(
	ctx context.Context, req *connect.Request[financev1.UpdateCategoryRequest],
) (*connect.Response[financev1.UpdateCategoryResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("category_id", msg.GetCategoryId())
	if err != nil {
		return nil, err
	}
	params := db.UpdateCategoryParams{ID: id, FamilyID: c.familyID}
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
	if msg.Icon != nil {
		icon := trimmed(msg.GetIcon())
		if err := checkText("icon", icon, maxIconRunes); err != nil {
			return nil, err
		}
		params.Icon = &icon
	}
	if msg.Archived != nil {
		archived := msg.GetArchived()
		params.Archived = &archived
	}

	row, err := h.q.UpdateCategory(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("category")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update category")
	}
	return connect.NewResponse(&financev1.UpdateCategoryResponse{
		Category: toProtoCategory(row),
	}), nil
}

func (h *Handler) MoveCategory(
	ctx context.Context, req *connect.Request[financev1.MoveCategoryRequest],
) (*connect.Response[financev1.MoveCategoryResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("category_id", req.Msg.GetCategoryId())
	if err != nil {
		return nil, err
	}
	target, err := requireUUID("target_group_id", req.Msg.GetTargetGroupId())
	if err != nil {
		return nil, err
	}
	if _, err := h.q.GetCategoryGroup(ctx, db.GetCategoryGroupParams{
		ID: target, FamilyID: c.familyID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("category group")
		}
		return nil, h.internal(ctx, err, "get category group")
	}

	row, err := h.q.MoveCategory(ctx, db.MoveCategoryParams{
		ID: id, FamilyID: c.familyID, GroupID: target,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("category")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "move category")
	}
	return connect.NewResponse(&financev1.MoveCategoryResponse{
		Category: toProtoCategory(row),
	}), nil
}

// DeleteCategory reassigns its transactions rather than orphaning them; a category with
// history and no reassignment target is refused, because a deleted category that takes a
// month of spend with it is data loss the user did not ask for.
func (h *Handler) DeleteCategory(
	ctx context.Context, req *connect.Request[financev1.DeleteCategoryRequest],
) (*connect.Response[financev1.DeleteCategoryResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("category_id", req.Msg.GetCategoryId())
	if err != nil {
		return nil, err
	}
	target, err := optionalUUID("reassign_to_category_id", req.Msg.GetReassignToCategoryId())
	if err != nil {
		return nil, err
	}
	if target.Valid && pgconv.UUIDString(target) == pgconv.UUIDString(id) {
		return nil, invalid("reassign_to_category_id must be a different category")
	}

	var moved int64
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if _, err := q.GetCategory(ctx, db.GetCategoryParams{ID: id, FamilyID: c.familyID}); err != nil {
			return err
		}
		count, err := q.CountCategoryTransactions(ctx, db.CountCategoryTransactionsParams{
			CategoryID: id, FamilyID: c.familyID,
		})
		if err != nil {
			return err
		}
		if count > 0 {
			if !target.Valid {
				return connect.NewError(connect.CodeFailedPrecondition,
					errors.New("this category still has transactions; choose a category to move them to"))
			}
			if _, err := q.GetCategory(ctx, db.GetCategoryParams{
				ID: target, FamilyID: c.familyID,
			}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return notFound("reassignment category")
				}
				return err
			}
			moved, err = q.MoveTransactionsToCategory(ctx, db.MoveTransactionsToCategoryParams{
				CategoryID: id, FamilyID: c.familyID, TargetCategoryID: target,
			})
			if err != nil {
				return err
			}
		}
		rows, err := q.DeleteCategory(ctx, db.DeleteCategoryParams{ID: id, FamilyID: c.familyID})
		if err != nil {
			return err
		}
		if rows == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("category")
	}
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			return nil, connectErr
		}
		return nil, h.internal(ctx, err, "delete category")
	}
	return connect.NewResponse(&financev1.DeleteCategoryResponse{
		MovedTransactions: int32(moved),
	}), nil
}

func (h *Handler) ReorderCategories(
	ctx context.Context, req *connect.Request[financev1.ReorderCategoriesRequest],
) (*connect.Response[financev1.ReorderCategoriesResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := checkBatch("category_ids_in_order", req.Msg.GetCategoryIdsInOrder()); err != nil {
		return nil, err
	}
	ids, err := uuidList("category_ids_in_order", req.Msg.GetCategoryIdsInOrder())
	if err != nil {
		return nil, err
	}
	// The grid being reordered belongs to one group, and the renumbering is confined to it:
	// an id from another group is refused rather than quietly renumbering that group too.
	groupID, err := requireUUID("group_id", req.Msg.GetGroupId())
	if err != nil {
		return nil, err
	}
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		for i, id := range ids {
			n, err := q.ReorderCategory(ctx, db.ReorderCategoryParams{
				ID: id, FamilyID: c.familyID, SortOrder: int32(i), GroupID: groupID,
			})
			if err != nil {
				return err
			}
			if n == 0 {
				return notFound("category")
			}
		}
		return nil
	})
	if err != nil {
		// A category from another group is the caller's mistake, not the server's: let the
		// NotFound out of the transaction instead of flattening it to Internal.
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			return nil, connectErr
		}
		return nil, h.internal(ctx, err, "reorder categories")
	}
	return connect.NewResponse(&financev1.ReorderCategoriesResponse{}), nil
}
