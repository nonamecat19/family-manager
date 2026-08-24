// Package handler implements recipes.v1.RecipesService over Connect (and gRPC, from the same
// type). Authorization lives here: the caller's identity and family_id come from the verified
// access token, never from the request body. A recipe belongs to the caller's family; the
// service never calls services/family to check — it trusts the family_id claim.
package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nnc/family-manager/libs/go/rpc"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	recipesv1 "github.com/nnc/family-manager/sdk/go/recipes/v1"
	"github.com/nnc/family-manager/services/recipes/db"
)

// pgtypeUUID is the uuid type sqlc generates; aliased so saveIngredientsAndSteps is readable.
type pgtypeUUID = pgtype.UUID

// ImageStore is the slice of libs/go/storage this service uses. Narrow on purpose, same as
// EventBus: tests pass a fake instead of standing up MinIO.
type ImageStore interface {
	Put(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) (string, error)
}

// Handler serves recipes.v1.RecipesService.
type Handler struct {
	q           db.Querier
	tx          Tx
	bus         EventBus
	images      ImageStore
	imageBucket string
	log         *slog.Logger
	now         func() time.Time
}

// Tx is the transaction boundary. Implemented by internal/store over a pgx pool, and by the
// fake store in tests.
//
// It takes a callback rather than returning a transaction handle so that there is no way to
// begin one and forget to finish it, and so the Querier bound to the transaction is the only
// one in scope while it is open.
type Tx interface {
	InTx(ctx context.Context, fn func(q db.Querier) error) error
}

// Options configures a Handler. Only Queries is required.
type Options struct {
	Queries db.Querier
	// Tx groups the writes that must not half-apply. Nil means every write runs on its own,
	// which is what a zero-valued Options in a test gets.
	Tx  Tx
	Bus EventBus
	// Images and ImageBucket back UploadRecipeImage. Leaving Images nil makes that one RPC
	// fail with Unimplemented instead of every other procedure refusing to start — a MinIO
	// outage shouldn't take down recipe reads and writes any more than a NATS outage does.
	Images      ImageStore
	ImageBucket string
	Log         *slog.Logger
	// Now is injected by tests so expiry is deterministic.
	Now func() time.Time
}

func New(opts Options) *Handler {
	h := &Handler{
		q:           opts.Queries,
		tx:          opts.Tx,
		bus:         opts.Bus,
		images:      opts.Images,
		imageBucket: opts.ImageBucket,
		log:         opts.Log,
		now:         opts.Now,
	}
	if h.log == nil {
		h.log = slog.Default()
	}
	if h.now == nil {
		h.now = time.Now
	}
	if h.tx == nil {
		h.tx = withoutTx{h.q}
	}
	if h.bus == nil {
		h.bus = noopBus{}
	}
	if h.images == nil {
		h.images = noopImages{}
	}
	return h
}

// noopImages lets the service run (and every non-image test pass) with no MinIO attached.
type noopImages struct{}

func (noopImages) Put(context.Context, string, string, io.Reader, int64, string) (string, error) {
	return "", errors.New("image storage not configured")
}

// familyOf resolves the caller's family_id from the access token claim. Every recipe is
// family-scoped; a caller without a family cannot create one.
func (h *Handler) familyOf(ctx context.Context) (string, error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return "", err
	}
	if claims.FamilyID == "" {
		return "", connect.NewError(connect.CodeFailedPrecondition,
			errors.New("caller belongs to no family"))
	}
	return claims.FamilyID, nil
}

// userOf resolves the caller's user id from the token.
func (h *Handler) userOf(ctx context.Context) (string, error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return "", err
	}
	if claims.UserID == "" {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("token has no subject"))
	}
	return claims.UserID, nil
}

/* ------------------------------------------------------------------ categories */

func (h *Handler) CreateCategory(
	ctx context.Context, req *connect.Request[recipesv1.CreateCategoryRequest],
) (*connect.Response[recipesv1.CreateCategoryResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	name := trimmed(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	famUUID, err := pgconv.UUID(familyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad family id: %w", err))
	}

	cat, err := h.q.CreateCategory(ctx, db.CreateCategoryParams{FamilyID: famUUID, Name: name})
	if err != nil {
		return nil, h.internal(ctx, err, "create category")
	}
	return connect.NewResponse(&recipesv1.CreateCategoryResponse{Category: toProtoCategory(cat)}), nil
}

func (h *Handler) ListCategories(
	ctx context.Context, _ *connect.Request[recipesv1.ListCategoriesRequest],
) (*connect.Response[recipesv1.ListCategoriesResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListCategories(ctx, pgconv.MustUUID(familyID))
	if err != nil {
		return nil, h.internal(ctx, err, "list categories")
	}
	out := make([]*recipesv1.Category, 0, len(rows))
	for _, c := range rows {
		out = append(out, toProtoCategory(c))
	}
	return connect.NewResponse(&recipesv1.ListCategoriesResponse{Categories: out}), nil
}

func (h *Handler) CreateSubcategory(
	ctx context.Context, req *connect.Request[recipesv1.CreateSubcategoryRequest],
) (*connect.Response[recipesv1.CreateSubcategoryResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	name := trimmed(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	catID, err := pgconv.UUID(req.Msg.GetCategoryId())
	if err != nil || !catID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category_id is required"))
	}
	famUUID, err := pgconv.UUID(familyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad family id: %w", err))
	}

	sub, err := h.q.CreateSubcategory(ctx, db.CreateSubcategoryParams{
		CategoryID: catID, FamilyID: famUUID, Name: name,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create subcategory")
	}
	return connect.NewResponse(&recipesv1.CreateSubcategoryResponse{Subcategory: toProtoSubcategory(sub)}), nil
}

func (h *Handler) ListSubcategories(
	ctx context.Context, req *connect.Request[recipesv1.ListSubcategoriesRequest],
) (*connect.Response[recipesv1.ListSubcategoriesResponse], error) {
	catID, err := pgconv.UUID(req.Msg.GetCategoryId())
	if err != nil || !catID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category_id is required"))
	}
	rows, err := h.q.ListSubcategories(ctx, catID)
	if err != nil {
		return nil, h.internal(ctx, err, "list subcategories")
	}
	out := make([]*recipesv1.Subcategory, 0, len(rows))
	for _, s := range rows {
		out = append(out, toProtoSubcategory(s))
	}
	return connect.NewResponse(&recipesv1.ListSubcategoriesResponse{Subcategories: out}), nil
}

/* ------------------------------------------------------------------ recipes */

func (h *Handler) CreateRecipe(
	ctx context.Context, req *connect.Request[recipesv1.CreateRecipeRequest],
) (*connect.Response[recipesv1.CreateRecipeResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := h.userOf(ctx)
	if err != nil {
		return nil, err
	}
	title := trimmed(req.Msg.GetTitle())
	if title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("title is required"))
	}
	if err := checkRecipeSize(title, req.Msg.GetDescription(), req.Msg.GetNotes(),
		len(req.Msg.GetIngredients()), len(req.Msg.GetSteps())); err != nil {
		return nil, err
	}
	servings := req.Msg.GetServings()
	if servings <= 0 {
		servings = 1
	}

	famUUID, err := pgconv.UUID(familyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad family id: %w", err))
	}
	userUUID, err := pgconv.UUID(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad user id: %w", err))
	}
	catID, subID, err := taxonomyIDs(req.Msg.GetCategoryId(), req.Msg.GetSubcategoryId())
	if err != nil {
		return nil, err
	}

	// The recipe row and its ingredients and steps are one thing. Written separately, a
	// failure partway through the inserts left a recipe with the first four of its nine
	// ingredients and no indication that the rest were missing — and the response the caller
	// got was an error, so nobody went looking for a recipe they thought had not been created.
	var r db.Recipe
	if err := h.tx.InTx(ctx, func(q db.Querier) error {
		var err error
		r, err = q.CreateRecipe(ctx, db.CreateRecipeParams{
			FamilyID:      famUUID,
			Title:         title,
			Description:   req.Msg.GetDescription(),
			CategoryID:    catID,
			SubcategoryID: subID,
			Servings:      servings,
			PrepSeconds:   req.Msg.GetPrepSeconds(),
			CookSeconds:   req.Msg.GetCookSeconds(),
			AuthorUserID:  userUUID,
			Notes:         req.Msg.GetNotes(),
			Rating:        clampRating(req.Msg.GetRating()),
			Kcal:          maxInt32(req.Msg.GetNutrition().GetKcal(), 0),
			ProteinG:      nonNegative(req.Msg.GetNutrition().GetProteinG()),
			FatG:          nonNegative(req.Msg.GetNutrition().GetFatG()),
			CarbsG:        nonNegative(req.Msg.GetNutrition().GetCarbsG()),
		})
		if err != nil {
			return h.internal(ctx, err, "create recipe")
		}
		return h.saveIngredientsAndSteps(ctx, q, r.ID, req.Msg.GetIngredients(), req.Msg.GetSteps())
	}); err != nil {
		return nil, err
	}

	ingredients, steps, err := h.loadIngredientsAndSteps(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	h.publish(ctx, events.SubjectRecipesRecipeCreated, &recipesv1.RecipeCreatedEvent{
		FamilyId:     familyID,
		RecipeId:     pgconv.UUIDString(r.ID),
		AuthorUserId: userID,
		OccurredAt:   h.timestamp(),
	})

	return connect.NewResponse(&recipesv1.CreateRecipeResponse{
		Recipe: toProtoRecipe(r, ingredients, steps),
	}), nil
}

func (h *Handler) GetRecipe(
	ctx context.Context, req *connect.Request[recipesv1.GetRecipeRequest],
) (*connect.Response[recipesv1.GetRecipeResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		// Don't leak that the recipe exists in another family — return NotFound.
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	ingredients, steps, err := h.loadIngredientsAndSteps(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&recipesv1.GetRecipeResponse{
		Recipe: toProtoRecipe(r, ingredients, steps),
	}), nil
}

func (h *Handler) ListRecipes(
	ctx context.Context, req *connect.Request[recipesv1.ListRecipesRequest],
) (*connect.Response[recipesv1.ListRecipesResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}

	// favorite_only used to take a separate query path, which meant favorites could not be
	// sorted or filtered like the rest of the cookbook. It is now one more predicate on the
	// single ListRecipes query; the caller's user id is only resolved when it is needed.
	var userUUID pgtype.UUID
	if req.Msg.GetFavoriteOnly() {
		userID, err := h.userOf(ctx)
		if err != nil {
			return nil, err
		}
		userUUID = pgconv.MustUUID(userID)
	}

	arg := db.ListRecipesParams{
		FamilyID:        pgconv.MustUUID(familyID),
		CategoryID:      pgconv.MustUUID(req.Msg.GetCategoryId()),
		SubcategoryID:   pgconv.MustUUID(req.Msg.GetSubcategoryId()),
		Search:          optionalText(req.Msg.GetSearch()),
		Ingredient:      optionalText(req.Msg.GetIngredient()),
		MinRating:       clampInt32(req.Msg.GetMinRating(), 0, 5),
		MaxTotalSeconds: max0(req.Msg.GetMaxTotalSeconds()),
		FavoriteOnly:    req.Msg.GetFavoriteOnly(),
		UserID:          userUUID,
		Sort:            sortKey(req.Msg.GetSort()),
	}
	rows, err := h.q.ListRecipes(ctx, arg)
	if err != nil {
		return nil, h.internal(ctx, err, "list recipes")
	}
	out := make([]*recipesv1.Recipe, 0, len(rows))
	for _, r := range rows {
		ingredients, steps, err := h.loadIngredientsAndSteps(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, toProtoRecipe(r, ingredients, steps))
	}
	return connect.NewResponse(&recipesv1.ListRecipesResponse{Recipes: out}), nil
}

// RateRecipe sets the family's verdict, 1..5, or 0 to clear it.
func (h *Handler) RateRecipe(
	ctx context.Context, req *connect.Request[recipesv1.RateRecipeRequest],
) (*connect.Response[recipesv1.RateRecipeResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}
	rating := req.Msg.GetRating()
	if rating < 0 || rating > 5 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rating must be 0..5"))
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	r, err = h.q.SetRecipeRating(ctx, db.SetRecipeRatingParams{ID: r.ID, Rating: int16(rating)})
	if err != nil {
		return nil, h.internal(ctx, err, "set rating")
	}
	ingredients, steps, err := h.loadIngredientsAndSteps(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	h.publish(ctx, events.SubjectRecipesRecipeUpdated, &recipesv1.RecipeUpdatedEvent{
		FamilyId:   familyID,
		RecipeId:   pgconv.UUIDString(r.ID),
		OccurredAt: h.timestamp(),
	})

	return connect.NewResponse(&recipesv1.RateRecipeResponse{
		Recipe: toProtoRecipe(r, ingredients, steps),
	}), nil
}

func (h *Handler) UpdateRecipe(
	ctx context.Context, req *connect.Request[recipesv1.UpdateRecipeRequest],
) (*connect.Response[recipesv1.UpdateRecipeResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}
	title := trimmed(req.Msg.GetTitle())
	if title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("title is required"))
	}
	if err := checkRecipeSize(title, req.Msg.GetDescription(), req.Msg.GetNotes(),
		len(req.Msg.GetIngredients()), len(req.Msg.GetSteps())); err != nil {
		return nil, err
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	catID, subID, err := taxonomyIDs(req.Msg.GetCategoryId(), req.Msg.GetSubcategoryId())
	if err != nil {
		return nil, err
	}

	n := nutritionUpdate(req.Msg.GetNutrition())

	// Ingredients and steps are replaced by deleting and reinserting, which is the write in
	// this service that most needed a transaction: the delete committed on its own, so a
	// failure during the reinsert left the recipe with no ingredients at all. The user's
	// edit failed and their recipe lost its contents.
	if err := h.tx.InTx(ctx, func(q db.Querier) error {
		var err error
		r, err = q.UpdateRecipe(ctx, db.UpdateRecipeParams{
			ID:            r.ID,
			Title:         title,
			Description:   req.Msg.GetDescription(),
			CategoryID:    catID,
			SubcategoryID: subID,
			Servings:      req.Msg.GetServings(),
			PrepSeconds:   req.Msg.GetPrepSeconds(),
			CookSeconds:   req.Msg.GetCookSeconds(),
			Notes:         req.Msg.GetNotes(),
			Rating:        clampRating(req.Msg.GetRating()),
			Kcal:          n.kcal,
			ProteinG:      n.proteinG,
			FatG:          n.fatG,
			CarbsG:        n.carbsG,
		})
		if err != nil {
			return h.internal(ctx, err, "update recipe")
		}
		if err := q.DeleteIngredients(ctx, r.ID); err != nil {
			return h.internal(ctx, err, "clear ingredients")
		}
		if err := q.DeleteSteps(ctx, r.ID); err != nil {
			return h.internal(ctx, err, "clear steps")
		}
		return h.saveIngredientsAndSteps(ctx, q, r.ID, req.Msg.GetIngredients(), req.Msg.GetSteps())
	}); err != nil {
		return nil, err
	}

	ingredients, steps, err := h.loadIngredientsAndSteps(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	h.publish(ctx, events.SubjectRecipesRecipeUpdated, &recipesv1.RecipeUpdatedEvent{
		FamilyId:   familyID,
		RecipeId:   pgconv.UUIDString(r.ID),
		OccurredAt: h.timestamp(),
	})

	return connect.NewResponse(&recipesv1.UpdateRecipeResponse{
		Recipe: toProtoRecipe(r, ingredients, steps),
	}), nil
}

func (h *Handler) DeleteRecipe(
	ctx context.Context, req *connect.Request[recipesv1.DeleteRecipeRequest],
) (*connect.Response[recipesv1.DeleteRecipeResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	rows, err := h.q.DeleteRecipe(ctx, recipeID)
	if err != nil {
		return nil, h.internal(ctx, err, "delete recipe")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	h.publish(ctx, events.SubjectRecipesRecipeDeleted, &recipesv1.RecipeDeletedEvent{
		FamilyId:   familyID,
		RecipeId:   pgconv.UUIDString(r.ID),
		OccurredAt: h.timestamp(),
	})

	return connect.NewResponse(&recipesv1.DeleteRecipeResponse{}), nil
}

// maxImageBytes caps an upload well below MinIO/Postgres limits — recipe photos are phone
// camera shots the client should already be compressing, not raw sensor dumps.
const maxImageBytes = 8 << 20 // 8 MiB

func (h *Handler) UploadRecipeImage(
	ctx context.Context, req *connect.Request[recipesv1.UploadRecipeImageRequest],
) (*connect.Response[recipesv1.UploadRecipeImageResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}
	contentType := req.Msg.GetContentType()
	ext, ok := imageExtensions[contentType]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unsupported content_type %q", contentType))
	}
	data := req.Msg.GetImageData()
	if len(data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("image_data is required"))
	}
	if len(data) > maxImageBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("image exceeds 8MB limit"))
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	// Keyed by family+recipe with no timestamp: re-uploading a photo overwrites the same
	// object instead of leaking the previous one in the bucket forever.
	key := fmt.Sprintf("%s/%s%s", familyID, pgconv.UUIDString(recipeID), ext)
	url, err := h.images.Put(ctx, h.imageBucket, key, bytes.NewReader(data), int64(len(data)), contentType)
	if err != nil {
		return nil, h.internal(ctx, err, "upload image")
	}

	r, err = h.q.UpdateRecipeImage(ctx, db.UpdateRecipeImageParams{ID: recipeID, ImageUrl: url})
	if err != nil {
		return nil, h.internal(ctx, err, "save image url")
	}

	ingredients, steps, err := h.loadIngredientsAndSteps(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	h.publish(ctx, events.SubjectRecipesRecipeUpdated, &recipesv1.RecipeUpdatedEvent{
		FamilyId:   familyID,
		RecipeId:   pgconv.UUIDString(r.ID),
		OccurredAt: h.timestamp(),
	})

	return connect.NewResponse(&recipesv1.UploadRecipeImageResponse{
		Recipe: toProtoRecipe(r, ingredients, steps),
	}), nil
}

var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

/* ------------------------------------------------------------------ favorites & comments */

func (h *Handler) ToggleFavorite(
	ctx context.Context, req *connect.Request[recipesv1.ToggleFavoriteRequest],
) (*connect.Response[recipesv1.ToggleFavoriteResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := h.userOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	userUUID := pgconv.MustUUID(userID)
	isFav, err := h.q.IsFavorite(ctx, db.IsFavoriteParams{RecipeID: recipeID, UserID: userUUID})
	if err != nil {
		return nil, h.internal(ctx, err, "check favorite")
	}

	if isFav {
		if err := h.q.RemoveFavorite(ctx, db.RemoveFavoriteParams{RecipeID: recipeID, UserID: userUUID}); err != nil {
			return nil, h.internal(ctx, err, "remove favorite")
		}
		if err := h.q.DecrementFavoriteCount(ctx, recipeID); err != nil {
			return nil, h.internal(ctx, err, "decrement count")
		}
		return connect.NewResponse(&recipesv1.ToggleFavoriteResponse{IsFavorite: false}), nil
	}

	if err := h.q.AddFavorite(ctx, db.AddFavoriteParams{RecipeID: recipeID, UserID: userUUID}); err != nil {
		return nil, h.internal(ctx, err, "add favorite")
	}
	if err := h.q.IncrementFavoriteCount(ctx, recipeID); err != nil {
		return nil, h.internal(ctx, err, "increment count")
	}
	return connect.NewResponse(&recipesv1.ToggleFavoriteResponse{IsFavorite: true}), nil
}

func (h *Handler) ListFavorites(
	ctx context.Context, _ *connect.Request[recipesv1.ListFavoritesRequest],
) (*connect.Response[recipesv1.ListFavoritesResponse], error) {
	userID, err := h.userOf(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListFavoriteRecipes(ctx, pgconv.MustUUID(userID))
	if err != nil {
		return nil, h.internal(ctx, err, "list favorites")
	}
	out := make([]*recipesv1.Recipe, 0, len(rows))
	for _, r := range rows {
		ingredients, steps, err := h.loadIngredientsAndSteps(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, toProtoRecipe(r, ingredients, steps))
	}
	return connect.NewResponse(&recipesv1.ListFavoritesResponse{Recipes: out}), nil
}

func (h *Handler) AddComment(
	ctx context.Context, req *connect.Request[recipesv1.AddCommentRequest],
) (*connect.Response[recipesv1.AddCommentResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := h.userOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}
	body := trimmed(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("body is required"))
	}
	if len([]rune(body)) > maxCommentRunes {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("a comment may be at most %d characters", maxCommentRunes))
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	userUUID := pgconv.MustUUID(userID)

	// comment_count is denormalised onto the recipe. Written outside a transaction, a failed
	// increment left a comment that exists and a count that does not know about it, and the
	// caller was told the whole thing failed — so the drift accumulated silently, and every
	// list view showed a number that was wrong and could never correct itself.
	var c db.RecipeComment
	if err := h.tx.InTx(ctx, func(q db.Querier) error {
		var err error
		c, err = q.AddComment(ctx, db.AddCommentParams{
			RecipeID: recipeID, UserID: userUUID, Body: body,
		})
		if err != nil {
			return h.internal(ctx, err, "add comment")
		}
		if err := q.IncrementCommentCount(ctx, recipeID); err != nil {
			return h.internal(ctx, err, "increment comment count")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return connect.NewResponse(&recipesv1.AddCommentResponse{Comment: toProtoComment(c)}), nil
}

func (h *Handler) ListComments(
	ctx context.Context, req *connect.Request[recipesv1.ListCommentsRequest],
) (*connect.Response[recipesv1.ListCommentsResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	rows, err := h.q.ListComments(ctx, recipeID)
	if err != nil {
		return nil, h.internal(ctx, err, "list comments")
	}
	out := make([]*recipesv1.Comment, 0, len(rows))
	for _, c := range rows {
		out = append(out, toProtoComment(c))
	}
	return connect.NewResponse(&recipesv1.ListCommentsResponse{Comments: out}), nil
}

/* ------------------------------------------------------------------ meal planning */

func (h *Handler) PlanMeal(
	ctx context.Context, req *connect.Request[recipesv1.PlanMealRequest],
) (*connect.Response[recipesv1.PlanMealResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	recipeID, err := pgconv.UUID(req.Msg.GetRecipeId())
	if err != nil || !recipeID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is required"))
	}
	date, err := pgconv.Date(req.Msg.GetDate())
	if err != nil || !date.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("date is required (YYYY-MM-DD)"))
	}

	r, err := h.q.GetRecipe(ctx, recipeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
		}
		return nil, h.internal(ctx, err, "get recipe")
	}
	if pgconv.UUIDString(r.FamilyID) != familyID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipe not found"))
	}

	famUUID := pgconv.MustUUID(familyID)
	entry, err := h.q.PlanMeal(ctx, db.PlanMealParams{
		FamilyID: famUUID,
		RecipeID: recipeID,
		PlanDate: date,
		Slot:     slotFromProto(req.Msg.GetSlot()),
		Servings: req.Msg.GetServings(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "plan meal")
	}

	h.publish(ctx, events.SubjectRecipesMealPlanned, &recipesv1.MealPlannedEvent{
		FamilyId:   familyID,
		EntryId:    pgconv.UUIDString(entry.ID),
		RecipeId:   pgconv.UUIDString(recipeID),
		Date:       req.Msg.GetDate(),
		Slot:       req.Msg.GetSlot(),
		OccurredAt: h.timestamp(),
	})

	return connect.NewResponse(&recipesv1.PlanMealResponse{Entry: toProtoMealPlanEntry(entry)}), nil
}

func (h *Handler) ListMealPlan(
	ctx context.Context, req *connect.Request[recipesv1.ListMealPlanRequest],
) (*connect.Response[recipesv1.ListMealPlanResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}

	fromDate, err := pgconv.Date(req.Msg.GetFromDate())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("from_date is invalid"))
	}
	if !fromDate.Valid {
		fromDate = pgconv.DateFromToday()
	}
	toDate, err := pgconv.Date(req.Msg.GetToDate())
	if err != nil || !toDate.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("to_date is required (YYYY-MM-DD)"))
	}

	rows, err := h.q.ListMealPlan(ctx, db.ListMealPlanParams{
		FamilyID:   pgconv.MustUUID(familyID),
		PlanDate:   fromDate,
		PlanDate_2: toDate,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list meal plan")
	}
	out := make([]*recipesv1.MealPlanEntry, 0, len(rows))
	for _, e := range rows {
		out = append(out, toProtoMealPlanEntry(e))
	}
	return connect.NewResponse(&recipesv1.ListMealPlanResponse{Entries: out}), nil
}

func (h *Handler) RemoveMealPlanEntry(
	ctx context.Context, req *connect.Request[recipesv1.RemoveMealPlanEntryRequest],
) (*connect.Response[recipesv1.RemoveMealPlanEntryResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	entryID, err := pgconv.UUID(req.Msg.GetEntryId())
	if err != nil || !entryID.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("entry_id is required"))
	}

	rows, err := h.q.RemoveMealPlanEntry(ctx, db.RemoveMealPlanEntryParams{
		ID: entryID, FamilyID: pgconv.MustUUID(familyID),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "remove meal plan entry")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("entry not found"))
	}
	return connect.NewResponse(&recipesv1.RemoveMealPlanEntryResponse{}), nil
}

func (h *Handler) TotalIngredients(
	ctx context.Context, req *connect.Request[recipesv1.TotalIngredientsRequest],
) (*connect.Response[recipesv1.TotalIngredientsResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}

	fromDate, err := pgconv.Date(req.Msg.GetFromDate())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("from_date is invalid"))
	}
	if !fromDate.Valid {
		fromDate = pgconv.DateFromToday()
	}
	toDate, err := pgconv.Date(req.Msg.GetToDate())
	if err != nil || !toDate.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("to_date is required (YYYY-MM-DD)"))
	}

	rows, err := h.q.TotalIngredients(ctx, db.TotalIngredientsParams{
		FamilyID:   pgconv.MustUUID(familyID),
		PlanDate:   fromDate,
		PlanDate_2: toDate,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "total ingredients")
	}
	out := make([]*recipesv1.IngredientTotal, 0, len(rows))
	for _, t := range rows {
		out = append(out, toProtoIngredientTotal(t))
	}
	return connect.NewResponse(&recipesv1.TotalIngredientsResponse{Totals: out}), nil
}

// maxBasketItems bounds an ad-hoc basket. The array is unnested into a join server-side, so
// an unbounded list is an unbounded query; nobody cooks 200 recipes off one shopping trip.
const maxBasketItems = 200

func (h *Handler) SumIngredients(
	ctx context.Context, req *connect.Request[recipesv1.SumIngredientsRequest],
) (*connect.Response[recipesv1.SumIngredientsResponse], error) {
	familyID, err := h.familyOf(ctx)
	if err != nil {
		return nil, err
	}
	items := req.Msg.GetItems()
	if len(items) == 0 {
		return connect.NewResponse(&recipesv1.SumIngredientsResponse{}), nil
	}
	if len(items) > maxBasketItems {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("at most %d basket items", maxBasketItems))
	}

	ids := make([]pgtype.UUID, 0, len(items))
	servings := make([]int32, 0, len(items))
	for _, it := range items {
		id, err := pgconv.UUID(it.GetRecipeId())
		if err != nil || !id.Valid {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("recipe_id is invalid"))
		}
		ids = append(ids, id)
		servings = append(servings, max0(it.GetServings()))
	}

	rows, err := h.q.SumIngredientsForBasket(ctx, db.SumIngredientsForBasketParams{
		RecipeIds:    ids,
		ServingsList: servings,
		FamilyID:     pgconv.MustUUID(familyID),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum ingredients")
	}
	out := make([]*recipesv1.IngredientTotal, 0, len(rows))
	for _, t := range rows {
		out = append(out, toProtoBasketTotal(t))
	}
	return connect.NewResponse(&recipesv1.SumIngredientsResponse{Totals: out}), nil
}

/* ------------------------------------------------------------------ internals */

// saveIngredientsAndSteps inserts ingredients and steps in order. Position is 1-based.
// Upper bounds on a recipe. None of these is a limit anyone will meet by cooking; they exist
// because a recipe is written by an authenticated household member into rows nobody prunes,
// and because saveIngredientsAndSteps does one INSERT per element. maxRequestBytes admits a
// 16 MiB body, which is room for hundreds of thousands of one-character ingredients — a single
// request that holds a pool connection for minutes and leaves the table that size afterwards.
//
// Runes, not bytes, throughout: a recipe written in Ukrainian must not be worth half as much
// text as the same recipe in English.
const (
	maxTitleRunes       = 200
	maxDescriptionRunes = 4000
	maxNotesRunes       = 4000
	maxIngredients      = 200
	maxSteps            = 200
	// A comment is a cooking note — "halve the sugar", "needs 10 more minutes" — not an
	// essay, and comments are the one thing here that grows without bound per recipe.
	maxCommentRunes = 2000
)

// checkRecipeSize enforces those bounds. It reports the first field that is too long rather
// than a list, because a client that has exceeded one of these has a bug, not a form to fix.
func checkRecipeSize(title, description, notes string, ingredients, steps int) error {
	tooLong := func(field string, value string, limit int) error {
		if len([]rune(value)) <= limit {
			return nil
		}
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("%s must be at most %d characters", field, limit))
	}
	if err := tooLong("title", title, maxTitleRunes); err != nil {
		return err
	}
	if err := tooLong("description", description, maxDescriptionRunes); err != nil {
		return err
	}
	if err := tooLong("notes", notes, maxNotesRunes); err != nil {
		return err
	}
	if ingredients > maxIngredients {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("a recipe may have at most %d ingredients", maxIngredients))
	}
	if steps > maxSteps {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("a recipe may have at most %d steps", maxSteps))
	}
	return nil
}

// taxonomyIDs parses the optional category and subcategory ids.
//
// Both were parsed with the error discarded, which turns "cat-1" — a client bug, a stale id
// after a category is deleted, a hand-written request — into an invalid pgtype.UUID. That is
// written as NULL, so the recipe is created or updated as uncategorised and the caller is told
// it succeeded. The recipe then does not appear under the category the user picked, and
// nothing anywhere recorded why.
//
// Empty stays empty: uncategorised is a legitimate state, just not one an unparseable id
// should reach by accident.
func taxonomyIDs(categoryID, subcategoryID string) (cat, sub pgtypeUUID, err error) {
	cat, err = pgconv.UUID(categoryID)
	if err != nil {
		return cat, sub, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("category_id is not a uuid: %w", err))
	}
	sub, err = pgconv.UUID(subcategoryID)
	if err != nil {
		return cat, sub, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("subcategory_id is not a uuid: %w", err))
	}
	return cat, sub, nil
}

// withoutTx is the fallback when no Tx is supplied: it runs the callback against the plain
// querier, so each statement commits on its own. It keeps a zero-valued Options working; it is
// not a transaction and does not pretend to be one.
type withoutTx struct{ q db.Querier }

func (w withoutTx) InTx(_ context.Context, fn func(db.Querier) error) error { return fn(w.q) }

// saveIngredientsAndSteps writes through the querier it is given, not through h.q: inside
// InTx that is the transaction-bound one, and using the handler's would silently run these
// inserts on a different connection outside the transaction.
func (h *Handler) saveIngredientsAndSteps(
	ctx context.Context, q db.Querier, recipeID pgtypeUUID,
	ingredients []*recipesv1.Ingredient, steps []*recipesv1.Step,
) error {
	for i, ing := range ingredients {
		name := trimmed(ing.GetName())
		if name == "" {
			continue
		}
		if err := q.AddIngredient(ctx, db.AddIngredientParams{
			RecipeID: recipeID,
			Position: int32(i + 1),
			Name:     name,
			Amount:   ing.GetAmount(),
			Unit:     ing.GetUnit(),
		}); err != nil {
			return h.internal(ctx, err, "add ingredient")
		}
	}
	for i, step := range steps {
		instr := trimmed(step.GetInstruction())
		if instr == "" {
			continue
		}
		if err := q.AddStep(ctx, db.AddStepParams{
			RecipeID:        recipeID,
			Position:        int32(i + 1),
			Instruction:     instr,
			DurationSeconds: step.GetDurationSeconds(),
		}); err != nil {
			return h.internal(ctx, err, "add step")
		}
	}
	return nil
}

func (h *Handler) loadIngredientsAndSteps(
	ctx context.Context, recipeID pgtypeUUID,
) ([]db.RecipeIngredient, []db.RecipeStep, error) {
	ingredients, err := h.q.ListIngredients(ctx, recipeID)
	if err != nil {
		return nil, nil, h.internal(ctx, err, "list ingredients")
	}
	steps, err := h.q.ListSteps(ctx, recipeID)
	if err != nil {
		return nil, nil, h.internal(ctx, err, "list steps")
	}
	return ingredients, steps, nil
}

// internal hides driver detail from clients while keeping the cause in the log.
// internal hands the cause to the log and an opaque reference to the caller, so a pgx error
// never becomes part of a response body.
func (h *Handler) internal(ctx context.Context, err error, what string) error {
	return rpc.Internal(ctx, h.log, err, what)
}
