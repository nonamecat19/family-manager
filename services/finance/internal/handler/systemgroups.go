package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/services/finance/db"
)

const (
	roleInvestments  = "investments"
	roleInstallments = "installments"
)

type systemGroup struct {
	name      string
	icon      string
	colorStep int32
}

var systemGroups = map[string]systemGroup{
	roleInvestments:  {name: "Інвестиції", icon: "trend-up", colorStep: 6},
	roleInstallments: {name: "Розстрочки", icon: "credit-card", colorStep: 7},
}

func ensureSystemGroup(ctx context.Context, q db.Querier, familyID pgtype.UUID, role string) (db.CategoryGroup, error) {
	spec := systemGroups[role]
	group, err := q.GetCategoryGroupByRole(ctx, db.GetCategoryGroupByRoleParams{FamilyID: familyID, Role: role})
	if err == nil || !errors.Is(err, pgx.ErrNoRows) {
		return group, err
	}
	group, err = q.AdoptCategoryGroupRole(ctx, db.AdoptCategoryGroupRoleParams{
		FamilyID: familyID, Role: role, Name: spec.name,
	})
	if err == nil || !errors.Is(err, pgx.ErrNoRows) {
		return group, err
	}
	group, err = q.CreateRoleCategoryGroup(ctx, db.CreateRoleCategoryGroupParams{
		FamilyID: familyID, Name: spec.name, Icon: spec.icon, ColorStep: spec.colorStep, Role: role,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return q.GetCategoryGroupByRole(ctx, db.GetCategoryGroupByRoleParams{FamilyID: familyID, Role: role})
	}
	return group, err
}

func createOwnedCategory(
	ctx context.Context, q db.Querier, familyID pgtype.UUID, role, name, icon string,
) (db.Category, error) {
	group, err := ensureSystemGroup(ctx, q, familyID, role)
	if err != nil {
		return db.Category{}, err
	}
	existing, err := q.ListCategories(ctx, db.ListCategoriesParams{
		FamilyID: familyID, GroupID: group.ID, IncludeArchived: true,
	})
	if err != nil {
		return db.Category{}, err
	}
	taken := make(map[string]bool, len(existing))
	for _, c := range existing {
		taken[strings.ToLower(strings.TrimSpace(c.Name))] = true
	}
	unique := name
	for n := 2; taken[strings.ToLower(unique)]; n++ {
		unique = fmt.Sprintf("%s %d", name, n)
	}
	return q.CreateCategory(ctx, db.CreateCategoryParams{
		FamilyID: familyID, GroupID: group.ID, Name: unique, Kind: kindExpense, Icon: icon,
	})
}

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

func categoryInUse() error {
	return connect.NewError(connect.CodeFailedPrecondition,
		errors.New("this category belongs to an investment or an installment; archive or delete that first"))
}

const privateInstallmentCategory = "Особиста розстрочка"

func (h *Handler) rejectOwnedCategory(ctx context.Context, id pgtype.UUID) error {
	owned, err := h.q.IsCategoryOwned(ctx, id)
	if err != nil {
		return h.internal(ctx, err, "check category owner")
	}
	if owned {
		return connect.NewError(connect.CodeFailedPrecondition,
			errors.New("this category belongs to an investment or an installment; edit it there"))
	}
	return nil
}
