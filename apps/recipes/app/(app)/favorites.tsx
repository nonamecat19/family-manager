import { useFavoriteRecipes } from "@fm/api";
import type { Recipe } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { Image, Pressable, ScrollView, Text, View } from "react-native";
import { initialOf, tintFor, Screen, ScreenHeader } from "@fm/ui";

import { useI18n } from "../../components/i18n/index.tsx";
import { formatDuration, metaLine } from "../../components/organic/format.ts";

export default function FavoritesScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const favorites = useFavoriteRecipes();
  const recipes = favorites.data ?? [];

  return (
    <Screen>
      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerClassName="gap-4.5 px-5.5 pb-7 pt-sm"
        refreshControl={undefined}
      >
        <ScreenHeader title={t("favorites.title")} onBack={() => router.back()} backLabel={t("common.back")} />

        <View className="flex-row flex-wrap gap-3">
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
          <Text className="font-fig text-15 leading-[22px] text-neutral-600">
            {t("favorites.empty")}
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
  const { t } = useI18n();
  const tint = tintFor(recipe.categoryId, index);
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={recipe.title}
      onPress={onPress}
      className="flex-1 basis-[45%] rounded-2xl bg-neutral-100 p-3.5 shadow-card"
    >
      <View
        className="h-[88px] items-center justify-center overflow-hidden rounded-xl"
        style={{ backgroundColor: tint.bg }}
      >
        {recipe.imageUrl !== "" ? (
          <Image source={{ uri: recipe.imageUrl }} className="h-[76px] w-[76px]" resizeMode="contain" />
        ) : (
          <Text className="font-cap text-26" style={{ color: tint.fg }}>
            {initialOf(recipe.title)}
          </Text>
        )}
      </View>
      <Text className="mt-2.5 font-cap text-15 leading-[17px]" numberOfLines={2}>
        {recipe.title}
      </Text>
      <Text className="mt-1.25 font-fig-bold text-12.5 text-neutral-600" numberOfLines={1}>
        {metaLine([
          formatDuration(recipe.prepSeconds + recipe.cookSeconds, t),
          recipe.rating > 0 ? `${recipe.rating}.0 ★` : undefined,
        ])}
      </Text>
    </Pressable>
  );
}
