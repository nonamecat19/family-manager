import { useRecipeCategories, useRecipeSubcategories, useRecipes } from "@fm/api";
import type { Recipe } from "@fm/sdk/recipes/v1/recipes_pb";
import { RecipeSort } from "@fm/sdk/recipes/v1/recipes_pb";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { FlatList, Image, Pressable, ScrollView, Text, TextInput, View } from "react-native";

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

const SORTS: { label: string; value: RecipeSort }[] = [
  { label: "Rating", value: RecipeSort.RATING },
  { label: "Time", value: RecipeSort.TIME },
  { label: "A–Z", value: RecipeSort.TITLE },
  { label: "Newest", value: RecipeSort.UNSPECIFIED },
];

const TIME_PRESETS: { label: string; seconds: number }[] = [
  { label: "Any", seconds: 0 },
  { label: "20 min", seconds: 1200 },
  { label: "45 min", seconds: 2700 },
  { label: "90 min", seconds: 5400 },
];

const RATING_PRESETS: { label: string; value: number }[] = [
  { label: "Any", value: 0 },
  { label: "3+", value: 3 },
  { label: "4+", value: 4 },
  { label: "5", value: 5 },
];

/**
 * The cookbook, browsable. Category comes in from Home (or the filter sheet), subcategory is
 * a chip row, and everything narrower than that lives in the sheet — so the list itself is
 * never more than a title, one row of chips and the recipes.
 */
export default function RecipeListScreen() {
  const router = useRouter();
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
                <Display size={28}>{category?.name ?? "All recipes"}</Display>
                <Text className="mt-[4px] font-fig-bold text-[13px] text-neutral-600">
                  {list.isPending ? "…" : `${recipes.length} recipe${recipes.length === 1 ? "" : "s"}`}
                </Text>
              </View>
              <Pressable
                accessibilityRole="button"
                accessibilityLabel="Filter"
                onPress={() => setFilterOpen(true)}
                className="flex-none flex-row items-center gap-[8px] rounded-full bg-accent px-[17px] py-[11px]"
              >
                <Icon name="filter" size={16} color="#ffffff" />
                <Text className="font-fig-bold text-[14px] text-white">
                  {activeFilters > 0 ? `Filter · ${activeFilters}` : "Filter"}
                </Text>
              </Pressable>
            </View>

            {subcategories.data && subcategories.data.length > 0 && (
              <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerClassName="gap-[8px] px-[22px] pb-[4px] pt-[16px]"
              >
                <Chip label="All" active={subcategoryId === ""} onPress={() => setSubcategoryId("")} tone="accent2" />
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

            <View className="flex-row items-center gap-[9px] px-[22px] pb-[4px] pt-[14px]">
              <Text className="font-fig-bold text-[13px] text-neutral-600">Sort</Text>
              {SORTS.map((option) => {
                const active = option.value === sort;
                return (
                  <Pressable
                    key={option.label}
                    accessibilityRole="button"
                    accessibilityLabel={`Sort by ${option.label}`}
                    accessibilityState={{ selected: active }}
                    onPress={() => setSort(option.value)}
                    className={`rounded-full px-[13px] py-[6px] ${active ? "bg-accent-200" : ""}`}
                  >
                    <Text
                      className={`font-fig-bold text-[13px] ${active ? "text-accent-800" : "text-neutral-600"}`}
                    >
                      {option.label}
                    </Text>
                  </Pressable>
                );
              })}
            </View>
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
              <Display size={20}>Nothing here yet</Display>
              <Text className="font-fig text-[15px] text-neutral-600">
                {activeFilters > 0
                  ? "No recipe matches those filters. Loosen one and try again."
                  : "Add the first recipe with the + button."}
              </Text>
            </View>
          )
        }
      />

      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Add recipe"
        onPress={() => router.push("/(app)/recipe-edit/new")}
        className="absolute bottom-[24px] right-[22px] h-[58px] w-[58px] items-center justify-center rounded-full bg-accent"
        style={{ boxShadow: "0 12px 32px rgba(46,43,37,0.22)" }}
      >
        <Icon name="plus" size={26} color="#ffffff" />
      </Pressable>

      <Sheet visible={filterOpen} onClose={() => setFilterOpen(false)} title="Filter">
        <Kicker className="mb-[10px]">Category</Kicker>
        <View className="mb-[20px] flex-row flex-wrap gap-[8px]">
          <Chip
            label="All"
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

        <Kicker className="mb-[10px]">Max time</Kicker>
        <View className="mb-[20px] flex-row flex-wrap gap-[8px]">
          {TIME_PRESETS.map((preset) => (
            <Chip
              key={preset.label}
              label={preset.label}
              active={preset.seconds === maxTotalSeconds}
              onPress={() => setMaxTotalSeconds(preset.seconds)}
              tone="accent2"
            />
          ))}
        </View>

        <Kicker className="mb-[10px]">Minimum rating</Kicker>
        <View className="mb-[20px] flex-row flex-wrap gap-[8px]">
          {RATING_PRESETS.map((preset) => (
            <Chip
              key={preset.label}
              label={preset.label}
              active={preset.value === minRating}
              onPress={() => setMinRating(preset.value)}
              tone="accent2"
            />
          ))}
          <Chip
            label="♥ Favourites"
            active={favoriteOnly}
            onPress={() => setFavoriteOnly(!favoriteOnly)}
            tone="accent2"
          />
        </View>

        <Kicker className="mb-[10px]">What can I cook with</Kicker>
        <TextInput
          accessibilityLabel="Ingredient"
          value={ingredient}
          onChangeText={setIngredient}
          placeholder="an ingredient you have"
          placeholderTextColor={organic.neutral[500]}
          className="mb-[24px] rounded-full border border-neutral-300 bg-neutral-100 px-[16px] py-[11px] font-fig text-[14px] text-fg"
        />

        <View className="flex-row gap-[10px]">
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Reset filters"
            onPress={resetFilters}
            className="flex-none rounded-full border-2 border-neutral-400 px-[24px] py-[13px]"
          >
            <Text className="font-fig-x text-[15px] text-neutral-700">Reset</Text>
          </Pressable>
          <PrimaryButton
            title={`Show ${recipes.length} recipe${recipes.length === 1 ? "" : "s"}`}
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
  const tint = tintFor(recipe.categoryId, index);
  const time = formatDuration(recipe.prepSeconds + recipe.cookSeconds);
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
            {recipe.servings} serving{recipe.servings === 1 ? "" : "s"}
          </Text>
          {time !== "" && <Text className="font-fig-bold text-[12.5px] text-neutral-700">{time}</Text>}
        </View>
      </View>
      <RatingMark rating={recipe.rating} />
    </Pressable>
  );
}
