import {
  useMealPlan,
  usePlanMeal,
  useRecipe,
  useRecipes,
  useRemoveMealPlanEntry,
  useSumIngredients,
} from "@fm/api";
import { MealSlot } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Alert, Pressable, ScrollView, Text, View } from "react-native";
import { Icon, initialOf, organic, tintFor, Avatar, Chip, DashedButton, Display, Kicker, PrimaryButton, Screen, SegTabs, Sheet, Stepper } from "@fm/ui";

import { useBasket } from "../../components/basket.tsx";
import { useI18n, type TranslationKey } from "../../components/i18n/index.tsx";
import { formatDuration, formatTotalTime, metaLine } from "../../components/organic/format.ts";
import { buildWeek } from "../../components/organic/week.ts";
import { weekRange } from "../../components/organic/week.ts";

const TABS = ["Basket", "Week"] as const;
type Tab = (typeof TABS)[number];

const SLOTS: { labelKey: TranslationKey; value: MealSlot }[] = [
  { labelKey: "mealPlan.slotBreakfast", value: MealSlot.BREAKFAST },
  { labelKey: "mealPlan.slotLunch", value: MealSlot.LUNCH },
  { labelKey: "mealPlan.slotDinner", value: MealSlot.DINNER },
  { labelKey: "mealPlan.slotSnack", value: MealSlot.SNACK },
  { labelKey: "mealPlan.slotDessert", value: MealSlot.DESSERT },
];

const SLOT_LABEL_KEYS: Record<number, TranslationKey> = {
  0: "mealPlan.slotMeal",
  1: "mealPlan.slotBreakfast",
  2: "mealPlan.slotLunch",
  3: "mealPlan.slotDinner",
  4: "mealPlan.slotSnack",
  5: "mealPlan.slotDessert",
};

export default function PlanScreen() {
  const { t } = useI18n();
  const [tab, setTab] = useState<Tab>("Basket");

  return (
    <Screen>
      <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="gap-4.5 px-5.5 pb-7 pt-sm">
        <Display size={28}>{t("mealPlan.title")}</Display>
        <SegTabs
          options={TABS}
          value={tab}
          onChange={setTab}
          labels={{ Basket: t("mealPlan.tabBasket"), Week: t("mealPlan.tabWeek") }}
        />
        {tab === "Basket" ? <BasketTab /> : <WeekTab />}
      </ScrollView>
    </Screen>
  );
}

function BasketTab() {
  const router = useRouter();
  const { t } = useI18n();
  const basket = useBasket();
  const totals = useSumIngredients(basket.items);
  const all = useRecipes();

  const recipes = all.data ?? [];
  const chosen = basket.items
    .map((item) => recipes.find((r) => r.id === item.recipeId))
    .filter((r): r is NonNullable<typeof r> => r !== undefined);
  const totalSeconds = chosen.reduce(
    (sum, r) => sum + (r.prepSeconds + r.cookSeconds) * basket.batchesOf(r.id),
    0,
  );
  const batches = basket.items.reduce((n, i) => n + basket.batchesOf(i.recipeId), 0);

  return (
    <View className="gap-2.5">
      {basket.items.map((item, index) => (
        <BasketRow key={item.recipeId} recipeId={item.recipeId} index={index} />
      ))}

      <DashedButton
        title={basket.items.length === 0 ? t("mealPlan.pickFirstRecipe") : t("mealPlan.addAnotherRecipe")}
        onPress={() => router.push("/(app)/recipes")}
      />

      {basket.items.length > 0 && (
        <View className="mt-xs rounded-2xl bg-accent2-200 px-5 py-4.5">
          <View className="flex-row justify-between">
            <Text className="font-fig-bold text-[14px] text-accent2-800">
              {t("mealPlan.recipesAndBatches", {
                recipes: t("plurals.recipesCount", { count: basket.items.length }),
                batches: t("plurals.batchesCount", { count: batches }),
              })}
            </Text>
            <Text className="font-fig-bold text-[14px] text-accent2-800">
              {formatTotalTime(totalSeconds, t)}
            </Text>
          </View>
          <PrimaryButton
            title={
              totals.data && totals.data.length > 0
                ? t("mealPlan.shoppingListItems", { count: totals.data.length })
                : t("mealPlan.generateShoppingList")
            }
            onPress={() => router.push("/(app)/basket")}
            className="mt-3.5"
          />
        </View>
      )}
    </View>
  );
}

function BasketRow({ recipeId, index }: { recipeId: string; index: number }) {
  const recipe = useRecipe(recipeId);
  const { t } = useI18n();
  const basket = useBasket();
  const r = recipe.data;
  const tint = tintFor(r?.categoryId, index);

  return (
    <View className="flex-row items-center gap-3.25 rounded-2xl bg-neutral-100 py-3 pl-3 pr-3.5">
      <Avatar initial={initialOf(r?.title ?? "?")} tint={tint} size={52} />
      <View className="min-w-0 flex-1">
        <Text className="font-fig-bold text-[15px] leading-[18px] text-fg" numberOfLines={1}>
          {r?.title ?? t("mealPlan.loading")}
        </Text>
        <Text className="mt-0.75 font-fig-semi text-[12.5px] text-neutral-600" numberOfLines={1}>
          {r
            ? metaLine([
                formatDuration(r.prepSeconds + r.cookSeconds, t),
                r.rating > 0 ? `${r.rating}.0 ★` : undefined,
              ])
            : ""}
        </Text>
      </View>
      {}
      <Stepper
        compact
        min={0}
        value={basket.batchesOf(recipeId)}
        onChange={(n) => basket.setBatches(recipeId, n)}
        label={r?.title ? t("mealPlan.batchesOf", { title: r.title }) : t("mealPlan.batchesOfRecipe")}
      />
    </View>
  );
}

function WeekTab() {
  const { t } = useI18n();
  const { from, to } = weekRange(new Date());
  const plan = useMealPlan(from, to);
  const recipes = useRecipes();
  const planMeal = usePlanMeal();
  const removeEntry = useRemoveMealPlanEntry();

  const [addingTo, setAddingTo] = useState<string | null>(null);
  const [slot, setSlot] = useState<MealSlot>(MealSlot.DINNER);

  const days = buildWeek(new Date());
  const entries = plan.data ?? [];

  return (
    <View className="gap-2.25">
      {days.map((day) => {
        const dayEntries = entries.filter((e) => e.date === day.iso);
        return (
          <View key={day.iso} className="flex-row items-stretch gap-3.25">
            <View className="w-[46px] flex-none items-center pt-3.25">
              <Text className={`font-cap text-[17px] ${day.isToday ? "text-accent-700" : "text-fg"}`}>
                {day.num}
              </Text>
              <Text className="mt-0.25 font-fig-x text-[11px] uppercase tracking-[1px] text-neutral-600">
                {t(day.dayKey)}
              </Text>
            </View>
            <View className="min-w-0 flex-1 gap-1.75">
              {dayEntries.map((entry) => {
                const recipe = recipes.data?.find((r) => r.id === entry.recipeId);
                const tint = tintFor(recipe?.categoryId);
                return (
                  <Pressable
                    key={entry.id}
                    accessibilityRole="button"
                    accessibilityLabel={t("mealPlan.removeMealFromDay", {
                      title: recipe?.title ?? t("mealPlan.unknownRecipe"),
                      day: t(day.dayKey),
                    })}
                    onPress={() =>
                      Alert.alert(t("mealPlan.removeFromWeekTitle"), recipe?.title ?? t("mealPlan.thisMeal"), [
                        { text: t("common.keep"), style: "cancel" },
                        {
                          text: t("common.remove"),
                          style: "destructive",
                          onPress: () => removeEntry.mutate(entry.id),
                        },
                      ])
                    }
                    className="flex-row justify-between gap-2.5 rounded-xl px-3.75 py-2.75"
                    style={{ backgroundColor: tint.bg }}
                  >
                    <Text
                      className="flex-1 font-fig-bold text-[14px]"
                      style={{ color: tint.fg }}
                      numberOfLines={1}
                    >
                      {recipe?.title ?? t("mealPlan.unknownRecipe")}
                    </Text>
                    <Text
                      className="font-fig-bold text-[12.5px] opacity-70"
                      style={{ color: tint.fg }}
                    >
                      {t(SLOT_LABEL_KEYS[entry.slot] ?? "mealPlan.slotMeal")}
                    </Text>
                  </Pressable>
                );
              })}
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={t("mealPlan.planAMealForDay", { day: t(day.dayKey), num: day.num })}
                onPress={() => setAddingTo(day.iso)}
                className="flex-row items-center justify-between rounded-xl border-[1.5px] border-dashed border-neutral-400 px-3.75 py-2.75"
              >
                <Text className="font-fig-bold text-[13.5px] text-neutral-500">
                  {dayEntries.length === 0 ? t("mealPlan.nothingPlanned") : t("mealPlan.addAnother")}
                </Text>
                <Icon name="plus" size={15} color={organic.neutral[500]} width={2.4} />
              </Pressable>
            </View>
          </View>
        );
      })}

      <Sheet visible={addingTo !== null} onClose={() => setAddingTo(null)} title={t("mealPlan.planAMeal")}>
        <Kicker className="mb-2.5">{t("mealPlan.slot")}</Kicker>
        <View className="mb-5 flex-row flex-wrap gap-sm">
          {SLOTS.map((s) => (
            <Chip key={s.labelKey} label={t(s.labelKey)} active={s.value === slot} onPress={() => setSlot(s.value)} />
          ))}
        </View>

        <Kicker className="mb-2.5">{t("mealPlan.recipe")}</Kicker>
        <ScrollView className="max-h-[280px]" showsVerticalScrollIndicator={false}>
          <View className="gap-sm">
            {(recipes.data ?? []).map((r, index) => (
              <Pressable
                key={r.id}
                accessibilityRole="button"
                accessibilityLabel={r.title}
                onPress={() => {
                  if (addingTo === null) return;
                  planMeal.mutate({ recipeId: r.id, date: addingTo, slot, servings: 0 });
                  setAddingTo(null);
                }}
                className="flex-row items-center gap-3 rounded-2xl bg-neutral-100 py-2.5 pl-2.5 pr-3.5"
              >
                <Avatar initial={initialOf(r.title)} tint={tintFor(r.categoryId, index)} size={40} />
                <Text className="flex-1 font-fig-bold text-[14.5px] text-fg" numberOfLines={1}>
                  {r.title}
                </Text>
              </Pressable>
            ))}
            {(recipes.data ?? []).length === 0 && (
              <Text className="font-fig text-[15px] text-neutral-600">
                {t("mealPlan.noRecipesYet")}
              </Text>
            )}
          </View>
        </ScrollView>
      </Sheet>
    </View>
  );
}
