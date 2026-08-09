package handler

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/services/recipes/db"
)

// fakeStore is an in-memory db.Querier. The handler's rules (family scoping, favorite toggle,
// ingredient totals) are what these tests exercise; Postgres itself is not under test here.
type fakeStore struct {
	mu sync.Mutex

	categories    map[string]db.RecipeCategory
	subcategories map[string]db.RecipeSubcategory
	recipes       map[string]db.Recipe
	ingredients   map[string][]db.RecipeIngredient // keyed by recipe_id
	steps         map[string][]db.RecipeStep
	favorites     map[string]bool                  // keyed recipe|user
	comments      map[string]db.RecipeComment
	mealPlan      map[string]db.MealPlanEntry

	failOn map[string]error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		categories:    map[string]db.RecipeCategory{},
		subcategories: map[string]db.RecipeSubcategory{},
		recipes:       map[string]db.Recipe{},
		ingredients:   map[string][]db.RecipeIngredient{},
		steps:         map[string][]db.RecipeStep{},
		favorites:     map[string]bool{},
		comments:      map[string]db.RecipeComment{},
		mealPlan:      map[string]db.MealPlanEntry{},
		failOn:        map[string]error{},
	}
}

func favKey(recipeID, userID pgtype.UUID) string {
	return pgconv.UUIDString(recipeID) + "|" + pgconv.UUIDString(userID)
}

func (s *fakeStore) fail(op string) error { return s.failOn[op] }

// --- categories ---

func (s *fakeStore) CreateCategory(_ context.Context, arg db.CreateCategoryParams) (db.RecipeCategory, error) {
	if err := s.fail("CreateCategory"); err != nil {
		return db.RecipeCategory{}, err
	}
	c := db.RecipeCategory{
		ID:        pgconv.MustUUID(newUUID()),
		FamilyID:  arg.FamilyID,
		Name:      arg.Name,
		CreatedAt: pgtype.Timestamptz{Valid: true},
	}
	s.categories[pgconv.UUIDString(c.ID)] = c
	return c, nil
}

func (s *fakeStore) ListCategories(_ context.Context, familyID pgtype.UUID) ([]db.RecipeCategory, error) {
	var out []db.RecipeCategory
	for _, c := range s.categories {
		if pgconv.UUIDString(c.FamilyID) == pgconv.UUIDString(familyID) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *fakeStore) CreateSubcategory(_ context.Context, arg db.CreateSubcategoryParams) (db.RecipeSubcategory, error) {
	sub := db.RecipeSubcategory{
		ID:         pgconv.MustUUID(newUUID()),
		CategoryID: arg.CategoryID,
		FamilyID:   arg.FamilyID,
		Name:       arg.Name,
		CreatedAt:  pgtype.Timestamptz{Valid: true},
	}
	s.subcategories[pgconv.UUIDString(sub.ID)] = sub
	return sub, nil
}

func (s *fakeStore) ListSubcategories(_ context.Context, categoryID pgtype.UUID) ([]db.RecipeSubcategory, error) {
	var out []db.RecipeSubcategory
	for _, sub := range s.subcategories {
		if pgconv.UUIDString(sub.CategoryID) == pgconv.UUIDString(categoryID) {
			out = append(out, sub)
		}
	}
	return out, nil
}

func (s *fakeStore) GetSubcategory(_ context.Context, id pgtype.UUID) (db.RecipeSubcategory, error) {
	sub, ok := s.subcategories[pgconv.UUIDString(id)]
	if !ok {
		return db.RecipeSubcategory{}, pgx.ErrNoRows
	}
	return sub, nil
}

// --- recipes ---

func (s *fakeStore) CreateRecipe(_ context.Context, arg db.CreateRecipeParams) (db.Recipe, error) {
	if err := s.fail("CreateRecipe"); err != nil {
		return db.Recipe{}, err
	}
	r := db.Recipe{
		ID:            pgconv.MustUUID(newUUID()),
		FamilyID:      arg.FamilyID,
		Title:         arg.Title,
		Description:   arg.Description,
		CategoryID:    arg.CategoryID,
		SubcategoryID: arg.SubcategoryID,
		Servings:      arg.Servings,
		PrepSeconds:   arg.PrepSeconds,
		CookSeconds:   arg.CookSeconds,
		AuthorUserID:  arg.AuthorUserID,
		CreatedAt:     pgtype.Timestamptz{Valid: true},
		UpdatedAt:     pgtype.Timestamptz{Valid: true},
	}
	s.recipes[pgconv.UUIDString(r.ID)] = r
	return r, nil
}

func (s *fakeStore) GetRecipe(_ context.Context, id pgtype.UUID) (db.Recipe, error) {
	r, ok := s.recipes[pgconv.UUIDString(id)]
	if !ok {
		return db.Recipe{}, pgx.ErrNoRows
	}
	return r, nil
}

func (s *fakeStore) ListRecipes(_ context.Context, arg db.ListRecipesParams) ([]db.Recipe, error) {
	var out []db.Recipe
	for _, r := range s.recipes {
		if pgconv.UUIDString(r.FamilyID) != pgconv.UUIDString(arg.FamilyID) {
			continue
		}
		if arg.CategoryID.Valid && pgconv.UUIDString(r.CategoryID) != pgconv.UUIDString(arg.CategoryID) {
			continue
		}
		if arg.SubcategoryID.Valid && pgconv.UUIDString(r.SubcategoryID) != pgconv.UUIDString(arg.SubcategoryID) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *fakeStore) ListFavoriteRecipes(_ context.Context, userID pgtype.UUID) ([]db.Recipe, error) {
	var out []db.Recipe
	for key, fav := range s.favorites {
		if !fav {
			continue
		}
		// key is recipeID|userID
		rid, uid := splitKey(key)
		if uid != pgconv.UUIDString(userID) {
			continue
		}
		if r, ok := s.recipes[rid]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *fakeStore) UpdateRecipe(_ context.Context, arg db.UpdateRecipeParams) (db.Recipe, error) {
	r, ok := s.recipes[pgconv.UUIDString(arg.ID)]
	if !ok {
		return db.Recipe{}, pgx.ErrNoRows
	}
	r.Title = arg.Title
	r.Description = arg.Description
	r.CategoryID = arg.CategoryID
	r.SubcategoryID = arg.SubcategoryID
	r.Servings = arg.Servings
	r.PrepSeconds = arg.PrepSeconds
	r.CookSeconds = arg.CookSeconds
	s.recipes[pgconv.UUIDString(r.ID)] = r
	return r, nil
}

func (s *fakeStore) UpdateRecipeImage(_ context.Context, arg db.UpdateRecipeImageParams) (db.Recipe, error) {
	r, ok := s.recipes[pgconv.UUIDString(arg.ID)]
	if !ok {
		return db.Recipe{}, pgx.ErrNoRows
	}
	r.ImageUrl = arg.ImageUrl
	s.recipes[pgconv.UUIDString(r.ID)] = r
	return r, nil
}

func (s *fakeStore) DeleteRecipe(_ context.Context, id pgtype.UUID) (int64, error) {
	key := pgconv.UUIDString(id)
	if _, ok := s.recipes[key]; !ok {
		return 0, nil
	}
	delete(s.recipes, key)
	delete(s.ingredients, key)
	delete(s.steps, key)
	return 1, nil
}

func (s *fakeStore) IncrementCommentCount(_ context.Context, id pgtype.UUID) error {
	r, ok := s.recipes[pgconv.UUIDString(id)]
	if ok {
		r.CommentCount++
		s.recipes[pgconv.UUIDString(id)] = r
	}
	return nil
}

func (s *fakeStore) AddIngredient(_ context.Context, arg db.AddIngredientParams) error {
	s.ingredients[pgconv.UUIDString(arg.RecipeID)] = append(
		s.ingredients[pgconv.UUIDString(arg.RecipeID)],
		db.RecipeIngredient{RecipeID: arg.RecipeID, Position: arg.Position, Name: arg.Name, Amount: arg.Amount, Unit: arg.Unit},
	)
	return nil
}

func (s *fakeStore) AddStep(_ context.Context, arg db.AddStepParams) error {
	s.steps[pgconv.UUIDString(arg.RecipeID)] = append(
		s.steps[pgconv.UUIDString(arg.RecipeID)],
		db.RecipeStep{RecipeID: arg.RecipeID, Position: arg.Position, Instruction: arg.Instruction, DurationSeconds: arg.DurationSeconds},
	)
	return nil
}

func (s *fakeStore) ListIngredients(_ context.Context, recipeID pgtype.UUID) ([]db.RecipeIngredient, error) {
	return s.ingredients[pgconv.UUIDString(recipeID)], nil
}

func (s *fakeStore) ListSteps(_ context.Context, recipeID pgtype.UUID) ([]db.RecipeStep, error) {
	return s.steps[pgconv.UUIDString(recipeID)], nil
}

func (s *fakeStore) DeleteIngredients(_ context.Context, recipeID pgtype.UUID) error {
	delete(s.ingredients, pgconv.UUIDString(recipeID))
	return nil
}

func (s *fakeStore) DeleteSteps(_ context.Context, recipeID pgtype.UUID) error {
	delete(s.steps, pgconv.UUIDString(recipeID))
	return nil
}

// --- favorites & comments ---

func (s *fakeStore) AddFavorite(_ context.Context, arg db.AddFavoriteParams) error {
	s.favorites[favKey(arg.RecipeID, arg.UserID)] = true
	return nil
}

func (s *fakeStore) RemoveFavorite(_ context.Context, arg db.RemoveFavoriteParams) error {
	delete(s.favorites, favKey(arg.RecipeID, arg.UserID))
	return nil
}

func (s *fakeStore) IsFavorite(_ context.Context, arg db.IsFavoriteParams) (bool, error) {
	return s.favorites[favKey(arg.RecipeID, arg.UserID)], nil
}

func (s *fakeStore) IncrementFavoriteCount(_ context.Context, id pgtype.UUID) error {
	r, ok := s.recipes[pgconv.UUIDString(id)]
	if ok {
		r.FavoriteCount++
		s.recipes[pgconv.UUIDString(id)] = r
	}
	return nil
}

func (s *fakeStore) DecrementFavoriteCount(_ context.Context, id pgtype.UUID) error {
	r, ok := s.recipes[pgconv.UUIDString(id)]
	if ok {
		r.FavoriteCount--
		s.recipes[pgconv.UUIDString(id)] = r
	}
	return nil
}

func (s *fakeStore) AddComment(_ context.Context, arg db.AddCommentParams) (db.RecipeComment, error) {
	c := db.RecipeComment{
		ID:        pgconv.MustUUID(newUUID()),
		RecipeID:  arg.RecipeID,
		UserID:    arg.UserID,
		Body:      arg.Body,
		CreatedAt: pgtype.Timestamptz{Valid: true},
	}
	s.comments[pgconv.UUIDString(c.ID)] = c
	return c, nil
}

func (s *fakeStore) ListComments(_ context.Context, recipeID pgtype.UUID) ([]db.RecipeComment, error) {
	var out []db.RecipeComment
	for _, c := range s.comments {
		if pgconv.UUIDString(c.RecipeID) == pgconv.UUIDString(recipeID) {
			out = append(out, c)
		}
	}
	return out, nil
}

// --- meal plan ---

func (s *fakeStore) PlanMeal(_ context.Context, arg db.PlanMealParams) (db.MealPlanEntry, error) {
	e := db.MealPlanEntry{
		ID:        pgconv.MustUUID(newUUID()),
		FamilyID:  arg.FamilyID,
		RecipeID:  arg.RecipeID,
		PlanDate:  arg.PlanDate,
		Slot:      arg.Slot,
		Servings:  arg.Servings,
		CreatedAt: pgtype.Timestamptz{Valid: true},
	}
	s.mealPlan[pgconv.UUIDString(e.ID)] = e
	return e, nil
}

func (s *fakeStore) ListMealPlan(_ context.Context, arg db.ListMealPlanParams) ([]db.MealPlanEntry, error) {
	var out []db.MealPlanEntry
	for _, e := range s.mealPlan {
		if pgconv.UUIDString(e.FamilyID) != pgconv.UUIDString(arg.FamilyID) {
			continue
		}
		// Simplified date comparison for tests
		out = append(out, e)
	}
	return out, nil
}

func (s *fakeStore) RemoveMealPlanEntry(_ context.Context, arg db.RemoveMealPlanEntryParams) (int64, error) {
	key := pgconv.UUIDString(arg.ID)
	if _, ok := s.mealPlan[key]; !ok {
		return 0, nil
	}
	if pgconv.UUIDString(s.mealPlan[key].FamilyID) != pgconv.UUIDString(arg.FamilyID) {
		return 0, nil
	}
	delete(s.mealPlan, key)
	return 1, nil
}

func (s *fakeStore) TotalIngredients(_ context.Context, arg db.TotalIngredientsParams) ([]db.TotalIngredientsRow, error) {
	// Simplified: sum ingredients across all meal plan entries in the family.
	totals := map[string]db.TotalIngredientsRow{}
	for _, e := range s.mealPlan {
		if pgconv.UUIDString(e.FamilyID) != pgconv.UUIDString(arg.FamilyID) {
			continue
		}
		ings := s.ingredients[pgconv.UUIDString(e.RecipeID)]
		r := s.recipes[pgconv.UUIDString(e.RecipeID)]
		scale := 1.0
		if e.Servings > 0 && r.Servings > 0 {
			scale = float64(e.Servings) / float64(r.Servings)
		}
		for _, ri := range ings {
			key := ri.Name + "|" + ri.Unit
			existing, ok := totals[key]
			if !ok {
				existing = db.TotalIngredientsRow{Name: ri.Name, Unit: ri.Unit, TotalAmount: "0"}
			}
			existing.TotalAmount = fmt.Sprintf("%.2f", parseFloat(existing.TotalAmount)+float64(len(ri.Name))*scale)
			totals[key] = existing
		}
	}
	var out []db.TotalIngredientsRow
	for _, t := range totals {
		out = append(out, t)
	}
	return out, nil
}

// --- helpers ---

type recorder struct {
	published []events.Subject
	err       error
}

func (r *recorder) Publish(_ context.Context, subject events.Subject, _ proto.Message) error {
	if r.err != nil {
		return r.err
	}
	r.published = append(r.published, subject)
	return nil
}

func (r *recorder) sawSubject(s events.Subject) bool {
	for _, got := range r.published {
		if got == s {
			return true
		}
	}
	return false
}

var errBoom = errors.New("boom")

var uuidCounter int

func newUUID() string {
	uuidCounter++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", uuidCounter)
}

func splitKey(key string) (string, string) {
	for i, c := range key {
		if c == '|' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}