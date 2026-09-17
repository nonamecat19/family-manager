import { useMealPlan, useRecipeCategories, useRecipes, useTotalIngredients } from "@fm/api";
import type { Category, Recipe } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { Image, Pressable, ScrollView, Text, View } from "react-native";
import { Icon, SearchIcon, initialOf, organic, tintFor, Avatar, Display, Kicker, RatingMark, Screen } from "@fm/ui";

import { useI18n } from "../../components/i18n/index.tsx";
import { formatDuration } from "../../components/organic/format.ts";
import { weekRange } from "../../components/organic/week.ts";

export default function HomeScreen() {
  const router = useRouter();
  const { t, locale } = useI18n();
  const all = useRecipes();
  const categories = useRecipeCategories();
  const { from, to } = weekRange(new Date());
  const plan = useMealPlan(from, to);
  const totals = useTotalIngredients(from, to);

  const recipes = all.data ?? [];
  const topRated = [...recipes].sort((a, b) => b.rating - a.rating).slice(0, 6);
  const planned = new Set((plan.data ?? []).map((e) => e.recipeId)).size;

  return (
    <Screen>
      <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="gap-5.5 px-5.5 pb-xl pt-sm">
        <View className="flex-row items-start justify-between gap-3">
          <View className="flex-1">
            <Kicker>{today(locale)}</Kicker>
            <Display size={33} className="mt-1.75">
              {t("home.title")}
            </Display>
          </View>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("home.yourProfile")}
            onPress={() => router.push("/(app)/settings")}
          >
            <Avatar initial="M" tint={{ bg: organic.accent2[300], fg: organic.accent2[800] }} size={48} />
          </Pressable>
        </View>

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={t("home.searchRecipes")}
          onPress={() => router.push("/(app)/search")}
          className="flex-row items-center gap-2.5 rounded-full border border-neutral-300 bg-neutral-100 px-4.5 py-3.25"
        >
          <SearchIcon />
          <Text className="font-fig text-16 text-neutral-600">
            {recipes.length > 0
              ? t("home.searchRecipesCount", { count: recipes.length })
              : t("home.searchRecipes")}
          </Text>
        </Pressable>

        {categories.data && categories.data.length > 0 && (
          <View className="flex-row flex-wrap gap-3">
            {categories.data.map((category, index) => (
              <CategoryCard
                key={category.id}
                category={category}
                index={index}
                count={recipes.filter((r) => r.categoryId === category.id).length}
                onPress={() => router.push(`/(app)/recipes?categoryId=${category.id}`)}
              />
            ))}
          </View>
        )}

        {topRated.length > 0 && (
          <>
            <View className="mt-0.5 flex-row items-baseline justify-between">
              <Display size={21}>{t("home.topRated")}</Display>
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={t("home.seeAllRecipes")}
                onPress={() => router.push("/(app)/recipes")}
              >
                <Text className="font-fig-bold text-14 text-accent-700">{t("home.seeAll")}</Text>
              </Pressable>
            </View>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              className="-mx-5.5 -mt-1.5"
              contentContainerClassName="gap-3.5 px-5.5 pb-sm pt-1.5"
            >
              {topRated.map((recipe, index) => (
                <TopRatedCard
                  key={recipe.id}
                  recipe={recipe}
                  index={index}
                  onPress={() => router.push(`/(app)/recipe/${recipe.id}`)}
                />
              ))}
            </ScrollView>
          </>
        )}

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={t("home.thisWeeksPlan")}
          onPress={() => router.push("/(app)/meal-plan")}
          className="flex-row items-center gap-lg rounded-2xl bg-accent2-200 px-5 py-4.5"
        >
          <View className="flex-1">
            <Display size={17} className="text-accent2-900">
              {t("home.thisWeeksPlan")}
            </Display>
            <Text className="mt-xs font-fig-semi text-13.5 text-accent2-800">
              {planned === 0
                ? t("home.nothingPlannedYet")
                : `${t("plurals.recipesCount", { count: planned })} · ${t("plurals.ingredientsToBuyCount", { count: (totals.data ?? []).length })}`}
            </Text>
          </View>
          <Icon name="forward" size={22} color={organic.accent2[800]} />
        </Pressable>

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={t("home.favorites")}
          onPress={() => router.push("/(app)/favorites")}
          className="flex-row items-center justify-between rounded-2xl border-2 border-dashed border-neutral-400 px-5 py-3.75"
        >
          <Text className="flex-1 font-fig-bold text-14.5 text-neutral-700" numberOfLines={2}>
            {t("home.keepComingBackTo")}
          </Text>
          <Icon name="forward" size={18} color={organic.neutral[700]} />
        </Pressable>
      </ScrollView>
    </Screen>
  );
}

function CategoryCard({
  category,
  index,
  count,
  onPress,
}: {
  category: Category;
  index: number;
  count: number;
  onPress: () => void;
}) {
  const { t } = useI18n();
  const tint = tintFor(category.name, index);
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={category.name}
      onPress={onPress}
      className="min-h-[100px] flex-1 basis-[45%] justify-between gap-xl rounded-2xl px-lg pb-4.25 pt-3.75"
      style={{ backgroundColor: tint.bg }}
    >
      <View className="h-[32px] w-[32px] items-center justify-center rounded-full bg-white/60">
        <Text className="font-cap text-14" style={{ color: tint.fg }}>
          {initialOf(category.name)}
        </Text>
      </View>
      <View>
        <Text className="font-cap text-18 leading-[20px]" style={{ color: tint.fg }}>
          {category.name}
        </Text>
        <Text className="mt-0.75 font-fig-bold text-12.5 opacity-70" style={{ color: tint.fg }}>
          {t("plurals.recipesCount", { count })}
        </Text>
      </View>
    </Pressable>
  );
}

function TopRatedCard({
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
  const time = formatDuration(recipe.prepSeconds + recipe.cookSeconds, t);
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={recipe.title}
      onPress={onPress}
      className="w-[158px] flex-none rounded-2xl bg-neutral-100 p-3.5 shadow-card"
    >
      <View
        className="h-[104px] items-center justify-center overflow-hidden rounded-xl"
        style={{ backgroundColor: tint.bg }}
      >
        {recipe.imageUrl !== "" ? (
          <Image source={{ uri: recipe.imageUrl }} className="h-[92px] w-[92px]" resizeMode="contain" />
        ) : (
          <Text className="font-cap text-32" style={{ color: tint.fg }}>
            {initialOf(recipe.title)}
          </Text>
        )}
      </View>
      <Text className="mt-2.75 font-cap text-15.5 leading-[18px]" numberOfLines={2}>
        {recipe.title}
      </Text>
      <View className="mt-1.75 flex-row items-center gap-sm">
        {time !== "" && <Text className="font-fig-bold text-12.5 text-neutral-700">{time}</Text>}
        {time !== "" && recipe.rating > 0 && <Text className="text-neutral-400">·</Text>}
        <RatingMark rating={recipe.rating} size={12} />
      </View>
    </Pressable>
  );
}

function today(locale: string): string {
  return new Date().toLocaleDateString(locale, {
    weekday: "long",
    day: "numeric",
    month: "long",
  });
}
