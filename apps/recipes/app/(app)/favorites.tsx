import { useFavoriteRecipes } from "@fm/api";
import type { Recipe } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { Image, Pressable, ScrollView, Text, View } from "react-native";

import { formatDuration, metaLine } from "../../components/organic/format.ts";
import { initialOf, tintFor } from "../../components/organic/tokens.ts";
import { Display, RoundButton, Screen } from "../../components/organic/ui.tsx";

/** Favourites is a wall of dishes, not a list of rows — you recognise these by sight. */
export default function FavoritesScreen() {
  const router = useRouter();
  const favorites = useFavoriteRecipes();
  const recipes = favorites.data ?? [];

  return (
    <Screen>
      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerClassName="gap-[18px] px-[22px] pb-[28px] pt-[8px]"
        refreshControl={undefined}
      >
        <View className="flex-row items-center gap-[12px]">
          <RoundButton icon="back" label="Back" onPress={() => router.back()} />
          <Display size={28}>Favourites</Display>
        </View>

        <View className="flex-row flex-wrap gap-[12px]">
          {recipes.map((recipe, index) => (
            <FavoriteCard
              key={recipe.id}
              recipe={recipe}
              index={index}
              onPress={() => router.push(`/(app)/recipe/${recipe.id}`)}
            />
          ))}
        </View>

        {!favorites.isPending && recipes.length === 0 && (
          <Text className="font-fig text-[15px] leading-[22px] text-neutral-600">
            Nothing saved yet. Tap the heart on a recipe and it will wait for you here.
          </Text>
        )}
      </ScrollView>
    </Screen>
  );
}

function FavoriteCard({
  recipe,
  index,
  onPress,
}: {
  recipe: Recipe;
  index: number;
  onPress: () => void;
}) {
  const tint = tintFor(recipe.categoryId, index);
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={recipe.title}
      onPress={onPress}
      className="flex-1 basis-[45%] rounded-2xl bg-neutral-100 p-[14px]"
      style={{ boxShadow: "0 1px 2px rgba(46,43,37,0.14)" }}
    >
      <View
        className="h-[88px] items-center justify-center overflow-hidden rounded-xl"
        style={{ backgroundColor: tint.bg }}
      >
        {recipe.imageUrl !== "" ? (
          <Image source={{ uri: recipe.imageUrl }} className="h-[76px] w-[76px]" resizeMode="contain" />
        ) : (
          <Text className="font-cap text-[26px]" style={{ color: tint.fg }}>
            {initialOf(recipe.title)}
          </Text>
        )}
      </View>
      <Text className="mt-[10px] font-cap text-[15px] leading-[17px]" numberOfLines={2}>
        {recipe.title}
      </Text>
      <Text className="mt-[5px] font-fig-bold text-[12.5px] text-neutral-600" numberOfLines={1}>
        {metaLine([
          formatDuration(recipe.prepSeconds + recipe.cookSeconds),
          recipe.rating > 0 ? `${recipe.rating}.0 ★` : undefined,
        ])}
      </Text>
    </Pressable>
  );
}
