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

const (
	slotBreakfast = "breakfast"
	slotLunch     = "lunch"
	slotDinner    = "dinner"
	slotSnack     = "snack"
	slotDessert   = "dessert"
)

type EventBus interface {
	Publish(ctx context.Context, subject events.Subject, msg proto.Message) error
}

type noopBus struct{}

func (noopBus) Publish(context.Context, events.Subject, proto.Message) error { return nil }

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

func toProtoNutrition(r db.Recipe) *recipesv1.Nutrition {
	return &recipesv1.Nutrition{
		Kcal:     r.Kcal,
		ProteinG: r.ProteinG,
		FatG:     r.FatG,
		CarbsG:   r.CarbsG,
	}
}

func nonNegative(v float32) float32 {
	if v < 0 {
		return 0
	}
	return v
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
		Notes:         r.Notes,
		Rating:        int32(r.Rating),
		ImageUrl:      r.ImageUrl,
		Nutrition:     toProtoNutrition(r),
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

func toProtoBasketTotal(t db.SumIngredientsForBasketRow) *recipesv1.IngredientTotal {
	return &recipesv1.IngredientTotal{
		Name:        t.Name,
		Unit:        t.Unit,
		TotalAmount: t.TotalAmount,
	}
}

func sortKey(s recipesv1.RecipeSort) string {
	switch s {
	case recipesv1.RecipeSort_RECIPE_SORT_TITLE:
		return "title"
	case recipesv1.RecipeSort_RECIPE_SORT_RATING:
		return "rating"
	case recipesv1.RecipeSort_RECIPE_SORT_TIME:
		return "time"
	case recipesv1.RecipeSort_RECIPE_SORT_FAVORITES:
		return "favorites"
	default:
		return "newest"
	}
}

func optionalText(v string) *string {
	t := trimmed(v)
	if t == "" {
		return nil
	}
	return &t
}

func clampInt32(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max0(v int32) int32 {
	if v < 0 {
		return 0
	}
	return v
}

func clampRating(v int32) int16 {
	return int16(clampInt32(v, 0, 5))
}

func maxInt32(v, lo int32) int32 {
	if v < lo {
		return lo
	}
	return v
}

type nutritionParams struct {
	kcal     *int32
	proteinG *float32
	fatG     *float32
	carbsG   *float32
}

func nutritionUpdate(n *recipesv1.Nutrition) nutritionParams {
	if n == nil {
		return nutritionParams{}
	}
	kcal := maxInt32(n.GetKcal(), 0)
	protein := nonNegative(n.GetProteinG())
	fat := nonNegative(n.GetFatG())
	carbs := nonNegative(n.GetCarbsG())
	return nutritionParams{kcal: &kcal, proteinG: &protein, fatG: &fat, carbsG: &carbs}
}
