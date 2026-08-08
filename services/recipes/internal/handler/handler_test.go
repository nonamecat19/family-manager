package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	recipesv1 "github.com/nnc/family-manager/sdk/go/recipes/v1"
	"github.com/nnc/family-manager/services/recipes/db"
)

// withClaims stamps a context with test claims — the auth interceptor is not in the call path
// for unit tests, so the handler reads claims from context directly.
func withClaims(ctx context.Context, userID, familyID string) context.Context {
	return fmauth.WithClaims(ctx, &fmauth.Claims{UserID: userID, FamilyID: familyID})
}

const (
	testUser   = "00000000-0000-4000-8000-000000000001"
	testFamily = "00000000-0000-4000-8000-000000000002"
)

func newTestHandler() (*Handler, *fakeStore, *recorder) {
	store := newFakeStore()
	rec := &recorder{}
	h := New(Options{Queries: store, Bus: rec})
	return h, store, rec
}

func TestCreateRecipe(t *testing.T) {
	h, store, rec := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)

	resp, err := h.CreateRecipe(ctx, connect.NewRequest(&recipesv1.CreateRecipeRequest{
		Title:       "Pancakes",
		Description: "Fluffy breakfast pancakes",
		Servings:    4,
		Ingredients: []*recipesv1.Ingredient{
			{Name: "flour", Amount: "200", Unit: "g"},
			{Name: "milk", Amount: "300", Unit: "ml"},
		},
		Steps: []*recipesv1.Step{
			{Position: 1, Instruction: "Mix flour and milk"},
			{Position: 2, Instruction: "Cook on a hot pan", DurationSeconds: 300},
		},
	}))
	if err != nil {
		t.Fatalf("CreateRecipe: %v", err)
	}
	if resp.Msg.Recipe.Title != "Pancakes" {
		t.Errorf("title = %q, want Pancakes", resp.Msg.Recipe.Title)
	}
	if len(resp.Msg.Recipe.Ingredients) != 2 {
		t.Errorf("ingredients = %d, want 2", len(resp.Msg.Recipe.Ingredients))
	}
	if len(resp.Msg.Recipe.Steps) != 2 {
		t.Errorf("steps = %d, want 2", len(resp.Msg.Recipe.Steps))
	}

	// Verify it was stored.
	recipeID := resp.Msg.Recipe.Id
	if _, ok := store.recipes[recipeID]; !ok {
		t.Error("recipe not stored")
	}
	if len(store.ingredients[recipeID]) != 2 {
		t.Errorf("stored ingredients = %d, want 2", len(store.ingredients[recipeID]))
	}

	// Verify event was published.
	if !rec.sawSubject("recipes.recipe.created") {
		t.Error("RecipeCreatedEvent not published")
	}
}

func TestCreateRecipeNoFamily(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, "") // no family

	_, err := h.CreateRecipe(ctx, connect.NewRequest(&recipesv1.CreateRecipeRequest{
		Title: "Test",
	}))
	if err == nil {
		t.Fatal("expected error for no family, got nil")
	}
}

func TestGetRecipeFamilyScoping(t *testing.T) {
	h, store, _ := newTestHandler()

	// Create a recipe in the test family.
	r := db.Recipe{
		ID:       pgconv.MustUUID(newUUID()),
		FamilyID: pgconv.MustUUID(testFamily),
		Title:    "Secret Cake",
	}
	store.recipes[pgconv.UUIDString(r.ID)] = r

	// A different family should get NotFound.
	ctx := withClaims(context.Background(), testUser, "00000000-0000-4000-8000-000000000099")
	_, err := h.GetRecipe(ctx, connect.NewRequest(&recipesv1.GetRecipeRequest{
		RecipeId: pgconv.UUIDString(r.ID),
	}))
	if err == nil {
		t.Fatal("expected NotFound for wrong family, got nil")
	}
}

func TestToggleFavorite(t *testing.T) {
	h, store, _ := newTestHandler()

	// Create a recipe directly in the store.
	r := db.Recipe{
		ID:       pgconv.MustUUID(newUUID()),
		FamilyID: pgconv.MustUUID(testFamily),
		Title:    "Fave",
	}
	store.recipes[pgconv.UUIDString(r.ID)] = r
	ctx := withClaims(context.Background(), testUser, testFamily)

	// Toggle on.
	resp, err := h.ToggleFavorite(ctx, connect.NewRequest(&recipesv1.ToggleFavoriteRequest{
		RecipeId: pgconv.UUIDString(r.ID),
	}))
	if err != nil {
		t.Fatalf("ToggleFavorite on: %v", err)
	}
	if !resp.Msg.IsFavorite {
		t.Error("expected is_favorite=true after first toggle")
	}

	// Toggle off.
	resp, err = h.ToggleFavorite(ctx, connect.NewRequest(&recipesv1.ToggleFavoriteRequest{
		RecipeId: pgconv.UUIDString(r.ID),
	}))
	if err != nil {
		t.Fatalf("ToggleFavorite off: %v", err)
	}
	if resp.Msg.IsFavorite {
		t.Error("expected is_favorite=false after second toggle")
	}
}

func TestAddComment(t *testing.T) {
	h, store, _ := newTestHandler()

	r := db.Recipe{
		ID:       pgconv.MustUUID(newUUID()),
		FamilyID: pgconv.MustUUID(testFamily),
		Title:    "Commented",
	}
	store.recipes[pgconv.UUIDString(r.ID)] = r
	ctx := withClaims(context.Background(), testUser, testFamily)

	resp, err := h.AddComment(ctx, connect.NewRequest(&recipesv1.AddCommentRequest{
		RecipeId: pgconv.UUIDString(r.ID),
		Body:     "Great recipe!",
	}))
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if resp.Msg.Comment.Body != "Great recipe!" {
		t.Errorf("body = %q", resp.Msg.Comment.Body)
	}

	// Comment count should be incremented.
	if store.recipes[pgconv.UUIDString(r.ID)].CommentCount != 1 {
		t.Errorf("comment_count = %d, want 1", store.recipes[pgconv.UUIDString(r.ID)].CommentCount)
	}
}

func TestPlanMeal(t *testing.T) {
	h, store, rec := newTestHandler()

	r := db.Recipe{
		ID:       pgconv.MustUUID(newUUID()),
		FamilyID: pgconv.MustUUID(testFamily),
		Title:    "Dinner",
		Servings: 2,
	}
	store.recipes[pgconv.UUIDString(r.ID)] = r
	ctx := withClaims(context.Background(), testUser, testFamily)

	resp, err := h.PlanMeal(ctx, connect.NewRequest(&recipesv1.PlanMealRequest{
		RecipeId: pgconv.UUIDString(r.ID),
		Date:     "2026-08-08",
		Slot:     recipesv1.MealSlot_MEAL_SLOT_DINNER,
		Servings: 4,
	}))
	if err != nil {
		t.Fatalf("PlanMeal: %v", err)
	}
	if resp.Msg.Entry.Slot != recipesv1.MealSlot_MEAL_SLOT_DINNER {
		t.Errorf("slot = %v", resp.Msg.Entry.Slot)
	}
	if resp.Msg.Entry.Servings != 4 {
		t.Errorf("servings = %d, want 4", resp.Msg.Entry.Servings)
	}

	if !rec.sawSubject("recipes.meal.planned") {
		t.Error("MealPlannedEvent not published")
	}
}

func TestDeleteRecipe(t *testing.T) {
	h, store, rec := newTestHandler()

	r := db.Recipe{
		ID:       pgconv.MustUUID(newUUID()),
		FamilyID: pgconv.MustUUID(testFamily),
		Title:    "ToDelete",
	}
	store.recipes[pgconv.UUIDString(r.ID)] = r
	ctx := withClaims(context.Background(), testUser, testFamily)

	_, err := h.DeleteRecipe(ctx, connect.NewRequest(&recipesv1.DeleteRecipeRequest{
		RecipeId: pgconv.UUIDString(r.ID),
	}))
	if err != nil {
		t.Fatalf("DeleteRecipe: %v", err)
	}

	if _, ok := store.recipes[pgconv.UUIDString(r.ID)]; ok {
		t.Error("recipe not deleted from store")
	}
	if !rec.sawSubject("recipes.recipe.deleted") {
		t.Error("RecipeDeletedEvent not published")
	}
}

func TestCreateCategory(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)

	resp, err := h.CreateCategory(ctx, connect.NewRequest(&recipesv1.CreateCategoryRequest{
		Name: "Dessert",
	}))
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	if resp.Msg.Category.Name != "Dessert" {
		t.Errorf("name = %q, want Dessert", resp.Msg.Category.Name)
	}
}