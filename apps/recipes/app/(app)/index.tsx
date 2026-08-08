import { useRecipes, useRecipeCategories } from "@fm/api";
import { Card, EmptyState, ErrorState, Loading } from "@fm/ui";
import type { Recipe } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { FlatList, Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

export default function RecipeListScreen() {
  const router = useRouter();
  const [selectedCategory, setSelectedCategory] = useState<string>("");

  const categories = useRecipeCategories();
  const list = useRecipes({ categoryId: selectedCategory || undefined });

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="p-lg">
        <Text className="text-display font-bold text-fg dark:text-fg-dark">Recipes</Text>
      </View>

      {/* Category filter — horizontal scroll */}
      {categories.data && categories.data.length > 0 && (
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerClassName="px-lg gap-sm pb-md"
        >
          <CategoryChip
            label="All"
            active={selectedCategory === ""}
            onPress={() => setSelectedCategory("")}
          />
          {categories.data.map((cat) => (
            <CategoryChip
              key={cat.id}
              label={cat.name}
              active={selectedCategory === cat.id}
              onPress={() => setSelectedCategory(cat.id)}
            />
          ))}
        </ScrollView>
      )}

      {list.isPending ? (
        <Loading />
      ) : list.isError ? (
        <ErrorState message={list.error.message} onRetry={() => void list.refetch()} />
      ) : (
        <FlatList
          data={list.data ?? []}
          keyExtractor={(r) => r.id}
          contentContainerClassName="px-lg pb-2xl gap-xs"
          renderItem={({ item }) => (
            <RecipeCard recipe={item} onPress={() => router.push(`/(app)/recipe/${item.id}`)} />
          )}
          ListEmptyComponent={
            <EmptyState
              title="No recipes yet"
              hint="Add your first recipe with the + button."
            />
          }
          refreshing={list.isRefetching}
          onRefresh={() => void list.refetch()}
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

function CategoryChip({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      className={`rounded-full px-md py-sm ${active ? "bg-primary" : "bg-card dark:bg-card-dark"}`}
    >
      <Text
        className={`text-body ${active ? "text-primary-fg" : "text-fg dark:text-fg-dark"}`}
      >
        {label}
      </Text>
    </Pressable>
  );
}

function RecipeCard({ recipe, onPress }: { recipe: Recipe; onPress: () => void }) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress}>
      <Card className="gap-xs p-md">
        <Text className="text-body font-semibold text-fg dark:text-fg-dark" numberOfLines={1}>
          {recipe.title}
        </Text>
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