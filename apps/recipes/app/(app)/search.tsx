import { useRecipes } from "@fm/api";
import type { Recipe } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { useI18n } from "../../components/i18n/index.tsx";
import { SearchIcon } from "../../components/organic/icons.tsx";
import { formatDuration, metaLine } from "../../components/organic/format.ts";
import { initialOf, organic, tintFor } from "../../components/organic/tokens.ts";
import { Avatar, Kicker, RoundButton, Screen } from "../../components/organic/ui.tsx";

/**
 * Search is its own screen rather than a field on the browse list, because "find me the
 * mushroom one" and "walk me through Desserts" are different intents — the design gives the
 * first a full-screen, keyboard-up surface and the second a drilldown.
 */
export default function SearchScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [recent, setRecent] = useState<string[]>([]);

  // One settle timer for the field: a keystroke must not be a request.
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 300);
    return () => clearTimeout(timer);
  }, [query]);

  // Recents are remembered for the session only — nothing on the wire stores them, and
  // persisting a search history locally is a decision for the family, not a side effect.
  useEffect(() => {
    if (debounced.length < 3) return;
    setRecent((prev) => [debounced, ...prev.filter((r) => r !== debounced)].slice(0, 6));
  }, [debounced]);

  const results = useRecipes({ search: debounced }, { enabled: debounced !== "" });
  const recipes = results.data ?? [];

  return (
    <Screen>
      <ScrollView
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
        contentContainerClassName="gap-[20px] px-[22px] pb-[24px] pt-[8px]"
      >
        <View className="flex-row items-center gap-[12px]">
          <RoundButton icon="back" label={t("common.back")} onPress={() => router.back()} />
          <View className="flex-1 flex-row items-center gap-[10px] rounded-full border-2 border-accent bg-neutral-100 px-[18px] py-[9px]">
            <SearchIcon size={18} color={organic.accent[700]} />
            <TextInput
              accessibilityLabel={t("search.placeholder")}
              autoFocus
              value={query}
              onChangeText={setQuery}
              placeholder={t("search.placeholder")}
              placeholderTextColor={organic.neutral[500]}
              returnKeyType="search"
              className="flex-1 font-fig-semi text-[16px] text-fg"
            />
          </View>
        </View>

        {recent.length > 0 && (
          <View>
            <Kicker className="mb-[11px]">{t("search.recent")}</Kicker>
            <View className="flex-row flex-wrap gap-[8px]">
              {recent.map((term) => (
                <Pressable
                  key={term}
                  accessibilityRole="button"
                  accessibilityLabel={t("search.searchFor", { term })}
                  onPress={() => setQuery(term)}
                  className="rounded-full border border-accent px-[10px] py-[4px]"
                >
                  <Text className="font-fig-semi text-[12px] text-accent">{term}</Text>
                </Pressable>
              ))}
            </View>
          </View>
        )}

        {debounced !== "" && (
          <View>
            <Kicker className="mb-[11px]">
              {results.isPending ? t("search.searching") : t("plurals.resultsCount", { count: recipes.length })}
            </Kicker>
            <View className="gap-[10px]">
              {recipes.map((recipe, index) => (
                <ResultRow
                  key={recipe.id}
                  recipe={recipe}
                  index={index}
                  onPress={() => router.push(`/(app)/recipe/${recipe.id}`)}
                />
              ))}
              {!results.isPending && recipes.length === 0 && (
                <Text className="font-fig text-[15px] text-neutral-600">
                  {t("search.noMatches", { query: debounced })}
                </Text>
              )}
            </View>
          </View>
        )}
      </ScrollView>
    </Screen>
  );
}

function ResultRow({
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
      className="flex-row items-center gap-[14px] rounded-2xl bg-neutral-100 py-[12px] pl-[12px] pr-[16px]"
    >
      <Avatar initial={initialOf(recipe.title)} tint={tint} size={52} />
      <View className="min-w-0 flex-1">
        <Text className="font-fig-bold text-[15.5px] text-fg" numberOfLines={1}>
          {recipe.title}
        </Text>
        <Text className="mt-[3px] font-fig-semi text-[12.5px] text-neutral-600" numberOfLines={1}>
          {metaLine([
            t("plurals.servingsCount", { count: recipe.servings }),
            formatDuration(recipe.prepSeconds + recipe.cookSeconds, t),
          ])}
        </Text>
      </View>
    </Pressable>
  );
}
