import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { MealSlot, RecipeSort } from "@fm/sdk/recipes/v1/recipes_pb";

import { useClients } from "./provider.tsx";
import { queryKeys } from "./queryKeys.ts";

/* ------------------------------------------------------------------ categories */

export function useRecipeCategories() {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.recipeCategories(),
    queryFn: async () => (await recipes.listCategories({})).categories,
  });
}

export function useCreateRecipeCategory() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => recipes.createCategory({ name }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

export function useRecipeSubcategories(categoryId: string) {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.recipeSubcategories(categoryId),
    queryFn: async () => (await recipes.listSubcategories({ categoryId })).subcategories,
    enabled: categoryId !== "",
  });
}

export function useCreateSubcategory() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { categoryId: string; name: string }) => recipes.createSubcategory(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

/* ------------------------------------------------------------------ recipes */

export interface RecipeListFilters {
  categoryId?: string;
  subcategoryId?: string;
  favoriteOnly?: boolean;
  search?: string;
  sort?: RecipeSort;
  /** 0..5; 0 means "don't filter on rating". */
  minRating?: number;
  /** prep + cook ceiling in seconds; 0 means "don't filter on time". */
  maxTotalSeconds?: number;
  /** substring match on an ingredient name — "what can I cook with chicken". */
  ingredient?: string;
}

export function useRecipes(filters: RecipeListFilters = {}, opts: { enabled?: boolean } = {}) {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.recipesList(filters),
    queryFn: async () => (await recipes.listRecipes(filters)).recipes,
    enabled: opts.enabled ?? true,
  });
}

export function useRecipe(id: string) {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.recipe(id),
    queryFn: async () => {
      const res = await recipes.getRecipe({ recipeId: id });
      return res.recipe ?? null;
    },
    enabled: id !== "",
  });
}

export interface IngredientInput {
  name: string;
  amount: string;
  unit: string;
}

export interface StepInput {
  position: number;
  instruction: string;
  durationSeconds: number;
}

export interface CreateRecipeInput {
  title: string;
  description: string;
  categoryId: string;
  subcategoryId: string;
  servings: number;
  prepSeconds: number;
  cookSeconds: number;
  ingredients: IngredientInput[];
  steps: StepInput[];
  notes: string;
  /** 1..5, or 0 for unrated. */
  rating: number;
  /**
   * Per-serving figures. Omitting this on an update leaves the stored macros alone rather
   * than zeroing them — the server COALESCEs it — so a caller that does not collect
   * nutrition cannot silently erase what a recipe was imported with.
   */
  nutrition?: { kcal: number; proteinG: number; fatG: number; carbsG: number };
}

export function useCreateRecipe() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateRecipeInput) => {
      const res = await recipes.createRecipe(input);
      return res.recipe;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

export function useUpdateRecipe() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateRecipeInput & { recipeId: string }) => recipes.updateRecipe(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

export function useDeleteRecipe() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (recipeId: string) => recipes.deleteRecipe({ recipeId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

export function useUploadRecipeImage() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { recipeId: string; imageData: Uint8Array; contentType: string }) => {
      const res = await recipes.uploadRecipeImage(input);
      return res.recipe;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

/**
 * useRateRecipe is the one-tap star control. It is separate from useUpdateRecipe because
 * rating from a list row must not require sending the recipe's whole body back.
 */
export function useRateRecipe() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { recipeId: string; rating: number }) => {
      const res = await recipes.rateRecipe(input);
      return res.recipe;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

/* ------------------------------------------------------------------ favorites */

export function useFavoriteRecipes() {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.favoriteRecipes(),
    queryFn: async () => (await recipes.listFavorites({})).recipes,
  });
}

export function useToggleFavorite() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (recipeId: string) => recipes.toggleFavorite({ recipeId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

/* ------------------------------------------------------------------ comments */

export function useComments(recipeId: string) {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.recipeComments(recipeId),
    queryFn: async () => (await recipes.listComments({ recipeId })).comments,
    enabled: recipeId !== "",
  });
}

export function useAddComment() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { recipeId: string; body: string }) => recipes.addComment(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

/* ------------------------------------------------------------------ meal plan */

export function useMealPlan(fromDate: string, toDate: string) {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.mealPlan(fromDate, toDate),
    queryFn: async () => (await recipes.listMealPlan({ fromDate, toDate })).entries,
    enabled: fromDate !== "" && toDate !== "",
  });
}

export function usePlanMeal() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { recipeId: string; date: string; slot: MealSlot; servings: number }) =>
      recipes.planMeal(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

export function useRemoveMealPlanEntry() {
  const { recipes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (entryId: string) => recipes.removeMealPlanEntry({ entryId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.recipes }),
  });
}

/* ------------------------------------------------------------------ totals */

export function useTotalIngredients(fromDate: string, toDate: string) {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.totalIngredients(fromDate, toDate),
    queryFn: async () => (await recipes.totalIngredients({ fromDate, toDate })).totals,
    enabled: fromDate !== "" && toDate !== "",
  });
}

/** One line of an ad-hoc cooking basket: a recipe and how many servings of it. */
export interface BasketItem {
  recipeId: string;
  /** 0 means "cook it as written" (the recipe's own servings). */
  servings: number;
}

/**
 * useSumIngredients totals an ad-hoc basket. It is a query, not a mutation, because nothing
 * is persisted — the basket lives in the app and the server just does the arithmetic, which
 * keeps the summing rules identical to the meal-plan totals.
 */
export function useSumIngredients(items: BasketItem[]) {
  const { recipes } = useClients();
  return useQuery({
    queryKey: queryKeys.sumIngredients(items),
    queryFn: async () => (await recipes.sumIngredients({ items })).totals,
    enabled: items.length > 0,
  });
}
