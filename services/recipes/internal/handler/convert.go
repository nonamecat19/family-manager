package handler

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	recipesv1 "github.com/nnc/family-manager/sdk/go/recipes/v1"
	"github.com/nnc/family-manager/services/recipes/db"
)

// Slot values as stored. The CHECK constraint in 000001_init.up.sql is the other half.
const (
	slotBreakfast = "breakfast"
	slotLunch     = "lunch"
	slotDinner    = "dinner"
	slotSnack     = "snack"
	slotDessert   = "dessert"
)

// EventBus is the slice of libs/go/events this service uses. Narrow on purpose: tests pass a
// recorder instead of standing up NATS.
type EventBus interface {
	Publish(ctx context.Context, subject events.Subject, msg proto.Message) error
}

// noopBus lets the service run (and tests pass) with no broker attached.
type noopBus struct{}

func (noopBus) Publish(context.Context, events.Subject, proto.Message) error { return nil }

// publish is fire-and-forget by design: a recipe change that succeeded must not be reported
// as failed because the broker hiccuped. The failure is logged, not returned.
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

func slotToProto(s string) recipesv1.MealSlot {
	switch s {
	case slotBreakfast:
		return recipesv1.MealSlot_MEAL_SLOT_BREAKFAST
	case slotLunch:
		return recipesv1.MealSlot_MEAL_SLOT_LUNCH
	case slotDinner:
		return recipesv1.MealSlot_MEAL_SLOT_DINNER
	case slotSnack:
		return recipesv1.MealSlot_MEAL_SLOT_SNACK
	case slotDessert:
		return recipesv1.MealSlot_MEAL_SLOT_DESSERT
	default:
		return recipesv1.MealSlot_MEAL_SLOT_UNSPECIFIED
	}
}

// slotFromProto defaults to lunch (the most common meal-plan slot). An unspecified slot must
// not silently become dessert.
func slotFromProto(s recipesv1.MealSlot) string {
	switch s {
	case recipesv1.MealSlot_MEAL_SLOT_BREAKFAST:
		return slotBreakfast
	case recipesv1.MealSlot_MEAL_SLOT_LUNCH:
		return slotLunch
	case recipesv1.MealSlot_MEAL_SLOT_DINNER:
		return slotDinner
	case recipesv1.MealSlot_MEAL_SLOT_SNACK:
		return slotSnack
	case recipesv1.MealSlot_MEAL_SLOT_DESSERT:
		return slotDessert
	default:
		return slotLunch
	}
}

func toProtoCategory(c db.RecipeCategory) *recipesv1.Category {
	return &recipesv1.Category{
		Id:        pgconv.UUIDString(c.ID),
		FamilyId:  pgconv.UUIDString(c.FamilyID),
		Name:      c.Name,
		CreatedAt: pgconv.Timestamp(c.CreatedAt),
	}
}

func toProtoSubcategory(s db.RecipeSubcategory) *recipesv1.Subcategory {
	return &recipesv1.Subcategory{
		Id:         pgconv.UUIDString(s.ID),
		CategoryId: pgconv.UUIDString(s.CategoryID),
		FamilyId:   pgconv.UUIDString(s.FamilyID),
		Name:       s.Name,
		CreatedAt:  pgconv.Timestamp(s.CreatedAt),
	}
}

func toProtoIngredient(ri db.RecipeIngredient) *recipesv1.Ingredient {
	return &recipesv1.Ingredient{
		Name:   ri.Name,
		Amount: ri.Amount,
		Unit:   ri.Unit,
	}
}

func toProtoStep(s db.RecipeStep) *recipesv1.Step {
	return &recipesv1.Step{
		Position:        s.Position,
		Instruction:     s.Instruction,
		DurationSeconds: s.DurationSeconds,
	}
}

func toProtoRecipe(
	r db.Recipe, ingredients []db.RecipeIngredient, steps []db.RecipeStep,
) *recipesv1.Recipe {
	ingProto := make([]*recipesv1.Ingredient, 0, len(ingredients))
	for _, ri := range ingredients {
		ingProto = append(ingProto, toProtoIngredient(ri))
	}
	stepProto := make([]*recipesv1.Step, 0, len(steps))
	for _, s := range steps {
		stepProto = append(stepProto, toProtoStep(s))
	}
	return &recipesv1.Recipe{
		Id:            pgconv.UUIDString(r.ID),
		FamilyId:      pgconv.UUIDString(r.FamilyID),
		Title:         r.Title,
		Description:   r.Description,
		CategoryId:    pgconv.UUIDString(r.CategoryID),
		SubcategoryId: pgconv.UUIDString(r.SubcategoryID),
		Servings:      r.Servings,
		PrepSeconds:   r.PrepSeconds,
		CookSeconds:   r.CookSeconds,
		Ingredients:   ingProto,
		Steps:         stepProto,
		AuthorUserId:  pgconv.UUIDString(r.AuthorUserID),
		FavoriteCount: r.FavoriteCount,
		CommentCount:  r.CommentCount,
		ImageUrl:      r.ImageUrl,
		CreatedAt:     pgconv.Timestamp(r.CreatedAt),
		UpdatedAt:     pgconv.Timestamp(r.UpdatedAt),
	}
}

func toProtoComment(c db.RecipeComment) *recipesv1.Comment {
	return &recipesv1.Comment{
		Id:        pgconv.UUIDString(c.ID),
		RecipeId:  pgconv.UUIDString(c.RecipeID),
		UserId:    pgconv.UUIDString(c.UserID),
		Body:      c.Body,
		CreatedAt: pgconv.Timestamp(c.CreatedAt),
	}
}

func toProtoMealPlanEntry(e db.MealPlanEntry) *recipesv1.MealPlanEntry {
	return &recipesv1.MealPlanEntry{
		Id:        pgconv.UUIDString(e.ID),
		FamilyId:  pgconv.UUIDString(e.FamilyID),
		RecipeId:  pgconv.UUIDString(e.RecipeID),
		Date:      pgconv.DateString(e.PlanDate),
		Slot:      slotToProto(e.Slot),
		Servings:  e.Servings,
		CreatedAt: pgconv.Timestamp(e.CreatedAt),
	}
}

func toProtoIngredientTotal(t db.TotalIngredientsRow) *recipesv1.IngredientTotal {
	return &recipesv1.IngredientTotal{
		Name:        t.Name,
		Unit:        t.Unit,
		TotalAmount: t.TotalAmount,
	}
}