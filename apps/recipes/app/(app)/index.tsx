import { useRecipes, useRecipeCategories, useRecipeSubcategories } from "@fm/api";
import { Card, EmptyState, ErrorState, Loading } from "@fm/ui";
import type { Category, Recipe, Subcategory } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { FlatList, Image, Pressable, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

/**
 * Browsing is category -> subcategory -> recipes, not a flat filtered list: a category (the
 * book/meal a recipe belongs to) can have a dozen subcategories, so showing every recipe as
 * soon as a category is picked just reproduces the same wall of cards one level down.
 */
export default function RecipeListScreen() {
  const router = useRouter();
  const [categoryId, setCategoryId] = useState("");
  const [subcategoryId, setSubcategoryId] = useState("");

  const categories = useRecipeCategories();
  const subcategories = useRecipeSubcategories(categoryId);
  // "all" is the sentinel for "every recipe in this category" (see SubcategoryBrowser) —
  // it is never a real subcategory id, so it must not reach the API as one.
  const list = useRecipes(
    {
      categoryId: categoryId || undefined,
      subcategoryId: subcategoryId !== "" && subcategoryId !== "all" ? subcategoryId : undefined,
    },
    { enabled: subcategoryId !== "" },
  );

  const selectedCategory = categories.data?.find((c) => c.id === categoryId);
  const selectedSubcategory = subcategories.data?.find((s) => s.id === subcategoryId);

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="flex-row items-center gap-sm p-lg">
        {categoryId !== "" && (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Back"
            onPress={() => (subcategoryId !== "" ? setSubcategoryId("") : setCategoryId(""))}
          >
            <Text className="text-title text-fg dark:text-fg-dark">‹</Text>
          </Pressable>
        )}
        <Text className="text-display font-bold text-fg dark:text-fg-dark">
          {selectedSubcategory?.name ?? selectedCategory?.name ?? "Recipes"}
        </Text>
      </View>

      {categoryId === "" ? (
        <CategoryBrowser
          isPending={categories.isPending}
          isError={categories.isError}
          error={categories.error}
          onRetry={() => void categories.refetch()}
          categories={categories.data ?? []}
          onSelect={setCategoryId}
        />
      ) : subcategoryId === "" ? (
        <SubcategoryBrowser
          isPending={subcategories.isPending}
          isError={subcategories.isError}
          error={subcategories.error}
          onRetry={() => void subcategories.refetch()}
          subcategories={subcategories.data ?? []}
          onSelect={setSubcategoryId}
          onSelectAll={() => setSubcategoryId("all")}
        />
      ) : (
        <RecipeList
          isPending={list.isPending}
          isError={list.isError}
          error={list.error}
          onRetry={() => void list.refetch()}
          recipes={list.data ?? []}
          isRefetching={list.isRefetching}
          onPressRecipe={(id) => router.push(`/(app)/recipe/${id}`)}
        />
      )}

      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Add recipe"
        onPress={() => router.push("/(app)/recipe-edit/new")}
        className="absolute bottom-xl right-xl h-14 w-14 items-center justify-center rounded-full bg-primary"
      >
        <Text className="text-title text-primary-fg">+</Text>
      </Pressable>
    </SafeAreaView>
  );
}

function CategoryBrowser({
  isPending,
  isError,
  error,
  onRetry,
  categories,
  onSelect,
}: {
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  onRetry: () => void;
  categories: Category[];
  onSelect: (id: string) => void;
}) {
  if (isPending) return <Loading />;
  if (isError) return <ErrorState message={error?.message ?? ""} onRetry={onRetry} />;
  return (
    <FlatList
      data={categories}
      keyExtractor={(c) => c.id}
      contentContainerClassName="px-lg pb-2xl gap-xs"
      renderItem={({ item }) => <BrowserRow label={item.name} onPress={() => onSelect(item.id)} />}
      ListEmptyComponent={
        <EmptyState title="No categories yet" hint="Add your first recipe with the + button." />
      }
    />
  );
}

// subcategoryId is set to the sentinel "all" (not "", which means "no category picked yet")
// when the caller wants every recipe in the category regardless of subcategory.
function SubcategoryBrowser({
  isPending,
  isError,
  error,
  onRetry,
  subcategories,
  onSelect,
  onSelectAll,
}: {
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  onRetry: () => void;
  subcategories: Subcategory[];
  onSelect: (id: string) => void;
  onSelectAll: () => void;
}) {
  if (isPending) return <Loading />;
  if (isError) return <ErrorState message={error?.message ?? ""} onRetry={onRetry} />;
  if (subcategories.length === 0) {
    // Nothing to drill into further — go straight to the recipe list for this category.
    onSelectAll();
    return <Loading />;
  }
  return (
    <FlatList
      data={subcategories}
      keyExtractor={(s) => s.id}
      contentContainerClassName="px-lg pb-2xl gap-xs"
      ListHeaderComponent={<BrowserRow label="All" onPress={onSelectAll} />}
      renderItem={({ item }) => <BrowserRow label={item.name} onPress={() => onSelect(item.id)} />}
    />
  );
}

function RecipeList({
  isPending,
  isError,
  error,
  onRetry,
  recipes,
  isRefetching,
  onPressRecipe,
}: {
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  onRetry: () => void;
  recipes: Recipe[];
  isRefetching: boolean;
  onPressRecipe: (id: string) => void;
}) {
  if (isPending) return <Loading />;
  if (isError) return <ErrorState message={error?.message ?? ""} onRetry={onRetry} />;
  return (
    <FlatList
      data={recipes}
      keyExtractor={(r) => r.id}
      contentContainerClassName="px-lg pb-2xl gap-xs"
      renderItem={({ item }) => <RecipeCard recipe={item} onPress={() => onPressRecipe(item.id)} />}
      ListEmptyComponent={
        <EmptyState title="No recipes yet" hint="Add your first recipe with the + button." />
      }
      refreshing={isRefetching}
      onRefresh={onRetry}
    />
  );
}

function BrowserRow({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress}>
      <Card className="flex-row items-center justify-between p-md">
        <Text className="text-body font-semibold text-fg dark:text-fg-dark">{label}</Text>
        <Text className="text-body text-muted dark:text-muted-dark">›</Text>
      </Card>
    </Pressable>
  );
}

function RecipeCard({ recipe, onPress }: { recipe: Recipe; onPress: () => void }) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress}>
      <Card className="gap-xs p-md">
        <View className="flex-row gap-md">
          {recipe.imageUrl !== "" && (
            <Image
              source={{ uri: recipe.imageUrl }}
              className="h-14 w-14 rounded-md bg-bg dark:bg-bg-dark"
              resizeMode="cover"
            />
          )}
          <View className="flex-1 gap-xs">
            <Text className="text-body font-semibold text-fg dark:text-fg-dark" numberOfLines={1}>
              {recipe.title}
            </Text>
          </View>
        </View>
        {recipe.description !== "" && (
          <Text className="text-caption text-muted dark:text-muted-dark" numberOfLines={2}>
            {recipe.description}
          </Text>
        )}
        <View className="flex-row gap-md">
          <Text className="text-caption text-muted dark:text-muted-dark">
            {recipe.servings} servings
          </Text>
          {(recipe.prepSeconds > 0 || recipe.cookSeconds > 0) && (
            <Text className="text-caption text-muted dark:text-muted-dark">
              {formatDuration(recipe.prepSeconds + recipe.cookSeconds)}
            </Text>
          )}
          {recipe.favoriteCount > 0 && (
            <Text className="text-caption text-muted dark:text-muted-dark">
              ♥ {recipe.favoriteCount}
            </Text>
          )}
          {recipe.commentCount > 0 && (
            <Text className="text-caption text-muted dark:text-muted-dark">
              💬 {recipe.commentCount}
            </Text>
          )}
        </View>
      </Card>
    </Pressable>
  );
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  if (mins < 60) return `${mins}m`;
  const hrs = Math.floor(mins / 60);
  return `${hrs}h ${mins % 60}m`;
}
