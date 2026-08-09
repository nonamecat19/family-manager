import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { MealSlot } from "@fm/sdk/recipes/v1/recipes_pb";

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