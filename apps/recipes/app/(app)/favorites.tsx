import { useFavoriteRecipes, useToggleFavorite } from "@fm/api";
import { Card, EmptyState, ErrorState, Loading } from "@fm/ui";
import type { Recipe } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { FlatList, Pressable, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

export default function FavoritesScreen() {
  const router = useRouter();
  const favorites = useFavoriteRecipes();
  const toggleFavorite = useToggleFavorite();

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="p-lg">
        <Text className="text-display font-bold text-fg dark:text-fg-dark">Favorites</Text>
      </View>

      {favorites.isPending ? (
        <Loading />
      ) : favorites.isError ? (
        <ErrorState message={favorites.error.message} onRetry={() => void favorites.refetch()} />
      ) : (
        <FlatList
          data={favorites.data ?? []}
          keyExtractor={(r) => r.id}
          contentContainerClassName="px-lg pb-2xl gap-xs"
          renderItem={({ item }) => (
            <FavoriteCard
              recipe={item}
              onPress={() => router.push(`/(app)/recipe/${item.id}`)}
              onUnfavorite={() => toggleFavorite.mutate(item.id)}
            />
          )}
          ListEmptyComponent={
            <EmptyState
              title="No favorites yet"
              hint="Tap the ♥ on a recipe to save it here."
            />
          }
          refreshing={favorites.isRefetching}
          onRefresh={() => void favorites.refetch()}
        />
      )}
    </SafeAreaView>
  );
}

function FavoriteCard({
  recipe,
  onPress,
  onUnfavorite,
}: {
  recipe: Recipe;
  onPress: () => void;
  onUnfavorite: () => void;
}) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress}>
      <Card className="flex-row items-center justify-between gap-md p-md">
        <View className="flex-1">
          <Text className="text-body font-semibold text-fg dark:text-fg-dark" numberOfLines={1}>
            {recipe.title}
          </Text>
          {recipe.description !== "" && (
            <Text className="text-caption text-muted dark:text-muted-dark" numberOfLines={1}>
              {recipe.description}
            </Text>
          )}
        </View>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Remove from favorites"
          onPress={onUnfavorite}
          className="h-10 w-10 items-center justify-center rounded-full bg-card dark:bg-card-dark"
        >
          <Text className="text-body text-error">♥</Text>
        </Pressable>
      </Card>
    </Pressable>
  );
}