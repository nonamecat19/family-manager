import { useRecipeCategories, useRecipeSubcategories, useRecipes } from "@fm/api";
import type { Recipe } from "@fm/sdk/recipes/v1/recipes_pb";
import { RecipeSort } from "@fm/sdk/recipes/v1/recipes_pb";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { FlatList, Image, Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { useI18n, type TranslationKey } from "../../components/i18n/index.tsx";
import { Icon } from "../../components/organic/icons.tsx";
import { formatDuration } from "../../components/organic/format.ts";
import { initialOf, organic, tintFor } from "../../components/organic/tokens.ts";
import {
  Avatar,
  Chip,
  Display,
  Kicker,
  PrimaryButton,
  RatingMark,
  Screen,
  Sheet,
} from "../../components/organic/ui.tsx";

const SORTS: { labelKey: TranslationKey; value: RecipeSort }[] = [
  { labelKey: "recipesList.sortRating", value: RecipeSort.RATING },
  { labelKey: "recipesList.sortTime", value: RecipeSort.TIME },
  { labelKey: "recipesList.sortTitle", value: RecipeSort.TITLE },
  { labelKey: "recipesList.sortNewest", value: RecipeSort.UNSPECIFIED },
];

const TIME_PRESETS: { labelKey: TranslationKey; seconds: number }[] = [
  { labelKey: "recipesList.timeAny", seconds: 0 },
  { labelKey: "recipesList.time20", seconds: 1200 },
  { labelKey: "recipesList.time45", seconds: 2700 },
  { labelKey: "recipesList.time90", seconds: 5400 },
];

const RATING_PRESETS: { labelKey: TranslationKey; value: number }[] = [
  { labelKey: "recipesList.ratingAny", value: 0 },
  { labelKey: "recipesList.rating3", value: 3 },
  { labelKey: "recipesList.rating4", value: 4 },
  { labelKey: "recipesList.rating5", value: 5 },
];

/**
 * The cookbook, browsable. Category comes in from Home (or the filter sheet), subcategory is
 * a chip row, and everything narrower than that lives in the sheet — so the list itself is
 * never more than a title, one row of chips and the recipes.
 */
export default function RecipeListScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const params = useLocalSearchParams<{ categoryId?: string }>();

  const linkedCategoryId = params.categoryId ?? "";
  const [categoryId, setCategoryId] = useState(linkedCategoryId);
  const [subcategoryId, setSubcategoryId] = useState("");
  const [lastLinked, setLastLinked] = useState(linkedCategoryId);
  const [sort, setSort] = useState<RecipeSort>(RecipeSort.RATING);
  const [minRating, setMinRating] = useState(0);
  const [maxTotalSeconds, setMaxTotalSeconds] = useState(0);
  const [favoriteOnly, setFavoriteOnly] = useState(false);
  const [ingredient, setIngredient] = useState("");
  const [debouncedIngredient, setDebouncedIngredient] = useState("");
  const [filterOpen, setFilterOpen] = useState(false);

  // Home links here with a category already chosen. Adjusting during render (rather than in
  // an effect) keeps the list from painting the old category for a frame — and comparing
  // against the last *link* rather than against `categoryId` means changing the category in
  // the filter sheet is not immediately undone.
  if (linkedCategoryId !== lastLinked) {
    setLastLinked(linkedCategoryId);
    setCategoryId(linkedCategoryId);
    setSubcategoryId("");
  }

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedIngredient(ingredient.trim()), 300);
    return () => clearTimeout(timer);
  }, [ingredient]);

  const categories = useRecipeCategories();
  const subcategories = useRecipeSubcategories(categoryId);
  const list = useRecipes({
    categoryId: categoryId || undefined,
    subcategoryId: subcategoryId || undefined,
    sort,
    minRating: minRating > 0 ? minRating : undefined,
    maxTotalSeconds: maxTotalSeconds > 0 ? maxTotalSeconds : undefined,
    favoriteOnly: favoriteOnly || undefined,
    ingredient: debouncedIngredient || undefined,
  });

  const category = categories.data?.find((c) => c.id === categoryId);
  const recipes = list.data ?? [];
  const activeFilters = [minRating > 0, maxTotalSeconds > 0, favoriteOnly, debouncedIngredient !== ""].filter(
    Boolean,
  ).length;

  function resetFilters() {
    setMinRating(0);
    setMaxTotalSeconds(0);
    setFavoriteOnly(false);
    setIngredient("");
    setDebouncedIngredient("");
  }

  return (
    <Screen>
      <FlatList
        data={recipes}
        keyExtractor={(r) => r.id}
        showsVerticalScrollIndicator={false}
        contentContainerClassName="gap-[10px] px-[22px] pb-[100px]"
        refreshing={list.isRefetching}
        onRefresh={() => void list.refetch()}
        ListHeaderComponent={
          <View className="-mx-[22px] pb-[12px]">
            <View className="flex-row items-center justify-between gap-[12px] px-[22px] pt-[8px]">
              <View className="flex-1">
                <Display size={28}>{category?.name ?? t("recipesList.allRecipes")}</Display>
                <Text className="mt-[4px] font-fig-bold text-[13px] text-neutral-600">
                  {list.isPending ? t("common.loadingEllipsis") : t("plurals.recipesCount", { count: recipes.length })}
                </Text>
              </View>
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={t("recipesList.filter")}
                onPress={() => setFilterOpen(true)}
                className="flex-none flex-row items-center gap-[8px] rounded-full bg-accent px-[17px] py-[11px]"
              >
                <Icon name="filter" size={16} color="#ffffff" />
                <Text className="font-fig-bold text-[14px] text-white">
                  {activeFilters > 0 ? t("recipesList.filterCount", { count: activeFilters }) : t("recipesList.filter")}
                </Text>
              </Pressable>
            </View>

            {subcategories.data && subcategories.data.length > 0 && (
              <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerClassName="gap-[8px] px-[22px] pb-[4px] pt-[16px]"
              >
                <Chip
                  label={t("common.all")}
                  active={subcategoryId === ""}
                  onPress={() => setSubcategoryId("")}
                  tone="accent2"
                />
                {subcategories.data.map((sub) => (
                  <Chip
                    key={sub.id}
                    label={sub.name}
                    active={subcategoryId === sub.id}
                    onPress={() => setSubcategoryId(subcategoryId === sub.id ? "" : sub.id)}
                    tone="accent2"
                  />
                ))}
              </ScrollView>
            )}

            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerClassName="flex-row items-center gap-[9px] px-[22px] pb-[4px] pt-[14px]"
            >
              <Text className="font-fig-bold text-[13px] text-neutral-600">{t("recipesList.sort")}</Text>
              {SORTS.map((option) => {
                const active = option.value === sort;
                const label = t(option.labelKey);
                return (
                  <Pressable
                    key={option.labelKey}
                    accessibilityRole="button"
                    accessibilityLabel={t("recipesList.sortBy", { label })}
                    accessibilityState={{ selected: active }}
                    onPress={() => setSort(option.value)}
                    className={`rounded-full px-[13px] py-[6px] ${active ? "bg-accent-200" : ""}`}
                  >
                    <Text
                      className={`font-fig-bold text-[13px] ${active ? "text-accent-800" : "text-neutral-600"}`}
                    >
                      {label}
                    </Text>
                  </Pressable>
                );
              })}
            </ScrollView>
          </View>
        }
        renderItem={({ item, index }) => (
          <RecipeRow
            recipe={item}
            index={index}
            onPress={() => router.push(`/(app)/recipe/${item.id}`)}
          />
        )}
        ListEmptyComponent={
          list.isPending ? null : (
            <View className="gap-[8px] pt-[40px]">
              <Display size={20}>{t("recipesList.nothingHereYet")}</Display>
              <Text className="font-fig text-[15px] text-neutral-600">
                {activeFilters > 0
                  ? t("recipesList.noneMatchFilters")
                  : t("recipesList.addFirstWithButton")}
              </Text>
            </View>
          )
        }
      />

      <Pressable
        accessibilityRole="button"
        accessibilityLabel={t("recipesList.addRecipe")}
        onPress={() => router.push("/(app)/recipe-edit/new")}
        className="absolute bottom-[24px] right-[22px] h-[58px] w-[58px] items-center justify-center rounded-full bg-accent"
        style={{ boxShadow: "0 12px 32px rgba(46,43,37,0.22)" }}
      >
        <Icon name="plus" size={26} color="#ffffff" />
      </Pressable>

      <Sheet visible={filterOpen} onClose={() => setFilterOpen(false)} title={t("recipesList.filterSheetTitle")}>
        <Kicker className="mb-[10px]">{t("recipesList.category")}</Kicker>
        <View className="mb-[20px] flex-row flex-wrap gap-[8px]">
          <Chip
            label={t("common.all")}
            active={categoryId === ""}
            onPress={() => {
              setCategoryId("");
              setSubcategoryId("");
            }}
          />
          {(categories.data ?? []).map((c) => (
            <Chip
              key={c.id}
              label={c.name}
              active={categoryId === c.id}
              onPress={() => {
                setCategoryId(c.id);
                setSubcategoryId("");
              }}
            />
          ))}
        </View>

        <Kicker className="mb-[10px]">{t("recipesList.maxTime")}</Kicker>
        <View className="mb-[20px] flex-row flex-wrap gap-[8px]">
          {TIME_PRESETS.map((preset) => (
            <Chip
              key={preset.labelKey}
              label={t(preset.labelKey)}
              active={preset.seconds === maxTotalSeconds}
              onPress={() => setMaxTotalSeconds(preset.seconds)}
              tone="accent2"
            />
          ))}
        </View>

        <Kicker className="mb-[10px]">{t("recipesList.minimumRating")}</Kicker>
        <View className="mb-[20px] flex-row flex-wrap gap-[8px]">
          {RATING_PRESETS.map((preset) => (
            <Chip
              key={preset.labelKey}
              label={t(preset.labelKey)}
              active={preset.value === minRating}
              onPress={() => setMinRating(preset.value)}
              tone="accent2"
            />
          ))}
          <Chip
            label={t("recipesList.favourites")}
            active={favoriteOnly}
            onPress={() => setFavoriteOnly(!favoriteOnly)}
            tone="accent2"
          />
        </View>

        <Kicker className="mb-[10px]">{t("recipesList.whatCanICookWith")}</Kicker>
        <TextInput
          accessibilityLabel={t("recipesList.ingredient")}
          value={ingredient}
          onChangeText={setIngredient}
          placeholder={t("recipesList.ingredientPlaceholder")}
          placeholderTextColor={organic.neutral[500]}
          className="mb-[24px] rounded-full border border-neutral-300 bg-neutral-100 px-[16px] py-[11px] font-fig text-[14px] text-fg"
        />

        <View className="flex-row gap-[10px]">
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("recipesList.resetFilters")}
            onPress={resetFilters}
            className="flex-none rounded-full border-2 border-neutral-400 px-[24px] py-[13px]"
          >
            <Text className="font-fig-x text-[15px] text-neutral-700">{t("recipesList.reset")}</Text>
          </Pressable>
          <PrimaryButton
            title={t("recipesList.showRecipes", { count: recipes.length })}
            onPress={() => setFilterOpen(false)}
            className="flex-1"
          />
        </View>
      </Sheet>
    </Screen>
  );
}

function RecipeRow({
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
      className="flex-row items-center gap-[14px] rounded-2xl bg-neutral-100 py-[12px] pl-[12px] pr-[16px]"
      style={{ boxShadow: "0 1px 2px rgba(46,43,37,0.14)" }}
    >
      {recipe.imageUrl !== "" ? (
        <Image
          source={{ uri: recipe.imageUrl }}
          className="h-[62px] w-[62px] flex-none rounded-full"
          style={{ backgroundColor: tint.bg }}
          resizeMode="contain"
        />
      ) : (
        <Avatar initial={initialOf(recipe.title)} tint={tint} size={62} />
      )}
      <View className="min-w-0 flex-1">
        <Text className="font-cap text-[16px] leading-[18px] text-fg" numberOfLines={2}>
          {recipe.title}
        </Text>
        <View className="mt-[6px] flex-row items-center gap-[7px]">
          <Text className="font-fig-bold text-[12.5px] text-neutral-700">
            {t("plurals.servingsCount", { count: recipe.servings })}
          </Text>
          {time !== "" && <Text className="font-fig-bold text-[12.5px] text-neutral-700">{time}</Text>}
        </View>
      </View>
      <RatingMark rating={recipe.rating} />
    </Pressable>
  );
}
