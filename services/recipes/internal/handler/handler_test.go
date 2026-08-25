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
// --- rating, filtering, sorting, ad-hoc basket -----------------------------

// seedRecipe creates a recipe through the handler so ingredients and steps land in the fake
// exactly as the real write path would leave them.
func seedRecipe(t *testing.T, h *Handler, ctx context.Context, req *recipesv1.CreateRecipeRequest) *recipesv1.Recipe {
	t.Helper()
	resp, err := h.CreateRecipe(ctx, connect.NewRequest(req))
	if err != nil {
		t.Fatalf("seed %q: %v", req.GetTitle(), err)
	}
	return resp.Msg.GetRecipe()
}

func TestRateRecipe(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)
	r := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{Title: "Borscht", Servings: 4})

	resp, err := h.RateRecipe(ctx, connect.NewRequest(&recipesv1.RateRecipeRequest{
		RecipeId: r.GetId(), Rating: 5,
	}))
	if err != nil {
		t.Fatalf("RateRecipe: %v", err)
	}
	if got := resp.Msg.GetRecipe().GetRating(); got != 5 {
		t.Errorf("rating = %d, want 5", got)
	}

	// 0 clears the rating rather than being rejected as out of range.
	resp, err = h.RateRecipe(ctx, connect.NewRequest(&recipesv1.RateRecipeRequest{
		RecipeId: r.GetId(), Rating: 0,
	}))
	if err != nil {
		t.Fatalf("RateRecipe clear: %v", err)
	}
	if got := resp.Msg.GetRecipe().GetRating(); got != 0 {
		t.Errorf("cleared rating = %d, want 0", got)
	}

	if _, err := h.RateRecipe(ctx, connect.NewRequest(&recipesv1.RateRecipeRequest{
		RecipeId: r.GetId(), Rating: 9,
	})); err == nil {
		t.Error("expected InvalidArgument for rating 9, got nil")
	}
}

func TestRateRecipeFamilyScoping(t *testing.T) {
	h, store, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)
	r := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{Title: "Not yours"})

	other := withClaims(context.Background(), testUser, "00000000-0000-4000-8000-0000000000ff")
	if _, err := h.RateRecipe(other, connect.NewRequest(&recipesv1.RateRecipeRequest{
		RecipeId: r.GetId(), Rating: 1,
	})); err == nil {
		t.Fatal("expected NotFound rating another family's recipe, got nil")
	}
	if got := store.recipes[r.GetId()].Rating; got != 0 {
		t.Errorf("rating was written across the family boundary: %d", got)
	}
}

func TestListRecipesFiltersAndSort(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)

	seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title: "Chicken WOK", Servings: 2, Rating: 5, PrepSeconds: 300, CookSeconds: 600,
		Ingredients: []*recipesv1.Ingredient{{Name: "chicken", Amount: "300", Unit: "g"}},
	})
	seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title: "Beef Stew", Servings: 4, Rating: 3, PrepSeconds: 600, CookSeconds: 7200,
		Ingredients: []*recipesv1.Ingredient{{Name: "beef", Amount: "500", Unit: "g"}},
	})
	seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title: "Apple Pie", Servings: 8, Rating: 0, PrepSeconds: 1200, CookSeconds: 2400,
		Ingredients: []*recipesv1.Ingredient{{Name: "apple", Amount: "4", Unit: "pcs"}},
	})

	titles := func(t *testing.T, req *recipesv1.ListRecipesRequest) []string {
		t.Helper()
		resp, err := h.ListRecipes(ctx, connect.NewRequest(req))
		if err != nil {
			t.Fatalf("ListRecipes: %v", err)
		}
		out := make([]string, 0, len(resp.Msg.GetRecipes()))
		for _, r := range resp.Msg.GetRecipes() {
			out = append(out, r.GetTitle())
		}
		return out
	}

	if got := titles(t, &recipesv1.ListRecipesRequest{Search: "wok"}); len(got) != 1 || got[0] != "Chicken WOK" {
		t.Errorf("search wok = %v, want [Chicken WOK]", got)
	}
	if got := titles(t, &recipesv1.ListRecipesRequest{Ingredient: "beef"}); len(got) != 1 || got[0] != "Beef Stew" {
		t.Errorf("ingredient beef = %v, want [Beef Stew]", got)
	}
	if got := titles(t, &recipesv1.ListRecipesRequest{MinRating: 4}); len(got) != 1 || got[0] != "Chicken WOK" {
		t.Errorf("min_rating 4 = %v, want [Chicken WOK]", got)
	}
	if got := titles(t, &recipesv1.ListRecipesRequest{MaxTotalSeconds: 1000}); len(got) != 1 || got[0] != "Chicken WOK" {
		t.Errorf("max_total_seconds 1000 = %v, want [Chicken WOK]", got)
	}

	wantTitle := []string{"Apple Pie", "Beef Stew", "Chicken WOK"}
	if got := titles(t, &recipesv1.ListRecipesRequest{Sort: recipesv1.RecipeSort_RECIPE_SORT_TITLE}); !equalStrings(got, wantTitle) {
		t.Errorf("sort title = %v, want %v", got, wantTitle)
	}
	wantRating := []string{"Chicken WOK", "Beef Stew", "Apple Pie"}
	if got := titles(t, &recipesv1.ListRecipesRequest{Sort: recipesv1.RecipeSort_RECIPE_SORT_RATING}); !equalStrings(got, wantRating) {
		t.Errorf("sort rating = %v, want %v", got, wantRating)
	}
	wantTime := []string{"Chicken WOK", "Apple Pie", "Beef Stew"}
	if got := titles(t, &recipesv1.ListRecipesRequest{Sort: recipesv1.RecipeSort_RECIPE_SORT_TIME}); !equalStrings(got, wantTime) {
		t.Errorf("sort time = %v, want %v", got, wantTime)
	}
}

func TestSumIngredientsBasket(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)

	wok := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title: "Chicken WOK", Servings: 2,
		Ingredients: []*recipesv1.Ingredient{
			{Name: "chicken", Amount: "300", Unit: "g"},
			{Name: "rice", Amount: "200", Unit: "g"},
		},
	})
	pilaf := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title: "Pilaf", Servings: 4,
		Ingredients: []*recipesv1.Ingredient{{Name: "rice", Amount: "400", Unit: "g"}},
	})

	resp, err := h.SumIngredients(ctx, connect.NewRequest(&recipesv1.SumIngredientsRequest{
		Items: []*recipesv1.RecipeQuantity{
			// 4 servings of a 2-serving recipe doubles it; 0 means "as written".
			{RecipeId: wok.GetId(), Servings: 4},
			{RecipeId: pilaf.GetId(), Servings: 0},
		},
	}))
	if err != nil {
		t.Fatalf("SumIngredients: %v", err)
	}
	got := map[string]string{}
	for _, tot := range resp.Msg.GetTotals() {
		got[tot.GetName()] = tot.GetTotalAmount()
	}
	if got["chicken"] != "600.00" {
		t.Errorf("chicken = %q, want 600.00", got["chicken"])
	}
	// 200 * 2 (scaled) + 400 (as written) — the two recipes collapse onto one line.
	if got["rice"] != "800.00" {
		t.Errorf("rice = %q, want 800.00", got["rice"])
	}
}

func TestSumIngredientsEmptyBasketIsNotAnError(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)

	resp, err := h.SumIngredients(ctx, connect.NewRequest(&recipesv1.SumIngredientsRequest{}))
	if err != nil {
		t.Fatalf("SumIngredients empty: %v", err)
	}
	if len(resp.Msg.GetTotals()) != 0 {
		t.Errorf("totals = %d, want 0", len(resp.Msg.GetTotals()))
	}
}

func TestSumIngredientsIgnoresOtherFamilies(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)
	mine := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title: "Mine", Servings: 1,
		Ingredients: []*recipesv1.Ingredient{{Name: "salt", Amount: "5", Unit: "g"}},
	})

	otherCtx := withClaims(context.Background(), testUser, "00000000-0000-4000-8000-0000000000ff")
	theirs := seedRecipe(t, h, otherCtx, &recipesv1.CreateRecipeRequest{
		Title: "Theirs", Servings: 1,
		Ingredients: []*recipesv1.Ingredient{{Name: "sugar", Amount: "5", Unit: "g"}},
	})

	resp, err := h.SumIngredients(ctx, connect.NewRequest(&recipesv1.SumIngredientsRequest{
		Items: []*recipesv1.RecipeQuantity{
			{RecipeId: mine.GetId()},
			{RecipeId: theirs.GetId()},
		},
	}))
	if err != nil {
		t.Fatalf("SumIngredients: %v", err)
	}
	for _, tot := range resp.Msg.GetTotals() {
		if tot.GetName() == "sugar" {
			t.Fatal("another family's ingredients leaked into the basket total")
		}
	}
}

func TestUpdateRecipeKeepsNotesAndRating(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)
	r := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title: "Soup", Servings: 2, Notes: "less salt", Rating: 4,
	})
	if r.GetNotes() != "less salt" || r.GetRating() != 4 {
		t.Fatalf("create dropped notes/rating: %q %d", r.GetNotes(), r.GetRating())
	}

	resp, err := h.UpdateRecipe(ctx, connect.NewRequest(&recipesv1.UpdateRecipeRequest{
		RecipeId: r.GetId(), Title: "Soup", Servings: 2, Notes: "more pepper", Rating: 2,
	}))
	if err != nil {
		t.Fatalf("UpdateRecipe: %v", err)
	}
	if resp.Msg.GetRecipe().GetNotes() != "more pepper" || resp.Msg.GetRecipe().GetRating() != 2 {
		t.Errorf("update = %q %d, want \"more pepper\" 2",
			resp.Msg.GetRecipe().GetNotes(), resp.Msg.GetRecipe().GetRating())
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Nutrition round-trips on create, and — the part that is easy to get wrong — survives an
// update that does not mention it. The edit screen predates these fields; if UpdateRecipe
// treated an absent Nutrition the way it treats an absent title, every save from that screen
// would silently erase the macros the recipe was imported with.
func TestNutritionCreateAndPreserveOnUpdate(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)

	r := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title:    "Кіноа з куркою",
		Servings: 1,
		Nutrition: &recipesv1.Nutrition{
			Kcal: 520, ProteinG: 48, FatG: 18.5, CarbsG: 42,
		},
	})
	if got := r.GetNutrition(); got.GetKcal() != 520 || got.GetProteinG() != 48 ||
		got.GetFatG() != 18.5 || got.GetCarbsG() != 42 {
		t.Fatalf("nutrition not stored on create: %+v", got)
	}

	// An update carrying no Nutrition at all must leave the figures untouched.
	upd, err := h.UpdateRecipe(ctx, connect.NewRequest(&recipesv1.UpdateRecipeRequest{
		RecipeId: r.GetId(), Title: "Кіноа з куркою", Servings: 2,
	}))
	if err != nil {
		t.Fatalf("UpdateRecipe: %v", err)
	}
	if got := upd.Msg.GetRecipe().GetNutrition(); got.GetKcal() != 520 || got.GetProteinG() != 48 {
		t.Fatalf("nutrition erased by an update that did not mention it: %+v", got)
	}
	if got := upd.Msg.GetRecipe().GetServings(); got != 2 {
		t.Fatalf("servings = %d, want the update to have applied", got)
	}

	// An update that does carry Nutrition overwrites it.
	upd, err = h.UpdateRecipe(ctx, connect.NewRequest(&recipesv1.UpdateRecipeRequest{
		RecipeId: r.GetId(), Title: "Кіноа з куркою", Servings: 2,
		Nutrition: &recipesv1.Nutrition{Kcal: 600, ProteinG: 50, FatG: 20, CarbsG: 45},
	}))
	if err != nil {
		t.Fatalf("UpdateRecipe: %v", err)
	}
	if got := upd.Msg.GetRecipe().GetNutrition(); got.GetKcal() != 600 || got.GetProteinG() != 50 {
		t.Fatalf("nutrition not overwritten when supplied: %+v", got)
	}
}

// Negative macros are floored rather than rejected: the CHECK constraint would otherwise turn
// a bad client value into a 500.
func TestNutritionNegativeValuesFloored(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)

	r := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{
		Title: "Nonsense", Servings: 1,
		Nutrition: &recipesv1.Nutrition{Kcal: -10, ProteinG: -1, FatG: -2, CarbsG: -3},
	})
	if got := r.GetNutrition(); got.GetKcal() != 0 || got.GetProteinG() != 0 ||
		got.GetFatG() != 0 || got.GetCarbsG() != 0 {
		t.Fatalf("negative nutrition not floored: %+v", got)
	}
}

// A recipe created without nutrition still reports a non-nil all-zero message, so the app can
// read the fields without a nil check.
func TestNutritionAlwaysPresentOnRead(t *testing.T) {
	h, _, _ := newTestHandler()
	ctx := withClaims(context.Background(), testUser, testFamily)

	r := seedRecipe(t, h, ctx, &recipesv1.CreateRecipeRequest{Title: "Plain", Servings: 1})
	if r.GetNutrition() == nil {
		t.Fatal("Nutrition is nil on a recipe created without it; want an all-zero message")
	}
	if r.GetNutrition().GetKcal() != 0 {
		t.Fatalf("kcal = %d, want 0", r.GetNutrition().GetKcal())
	}
}
