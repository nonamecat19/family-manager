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

import { useBasket } from "../../components/basket.tsx";
import { Icon } from "../../components/organic/icons.tsx";
import { formatDuration, formatTotalTime, metaLine } from "../../components/organic/format.ts";
import { initialOf, organic, tintFor } from "../../components/organic/tokens.ts";
import { buildWeek } from "../../components/organic/week.ts";
import { weekRange } from "../../components/organic/week.ts";
import {
  Avatar,
  Chip,
  DashedButton,
  Display,
  Kicker,
  PrimaryButton,
  Screen,
  SegTabs,
  Sheet,
  Stepper,
} from "../../components/organic/ui.tsx";

const TABS = ["Basket", "Week"] as const;
type Tab = (typeof TABS)[number];

const SLOTS: { label: string; value: MealSlot }[] = [
  { label: "Breakfast", value: MealSlot.BREAKFAST },
  { label: "Lunch", value: MealSlot.LUNCH },
  { label: "Dinner", value: MealSlot.DINNER },
  { label: "Snack", value: MealSlot.SNACK },
  { label: "Dessert", value: MealSlot.DESSERT },
];

const SLOT_LABELS: Record<number, string> = {
  0: "Meal",
  1: "Breakfast",
  2: "Lunch",
  3: "Dinner",
  4: "Snack",
  5: "Dessert",
};

/**
 * Plan holds the two halves of "what are we eating": an ad-hoc basket of recipes you intend
 * to cook (which becomes the shopping list), and the week itself. They share a tab because
 * they answer the same question at different resolutions.
 */
export default function PlanScreen() {
  const [tab, setTab] = useState<Tab>("Basket");

  return (
    <Screen>
      <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="gap-[18px] px-[22px] pb-[28px] pt-[8px]">
        <Display size={28}>Plan</Display>
        <SegTabs options={TABS} value={tab} onChange={setTab} />
        {tab === "Basket" ? <BasketTab /> : <WeekTab />}
      </ScrollView>
    </Screen>
  );
}

function BasketTab() {
  const router = useRouter();
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
    <View className="gap-[10px]">
      {basket.items.map((item, index) => (
        <BasketRow key={item.recipeId} recipeId={item.recipeId} index={index} />
      ))}

      <DashedButton
        title={basket.items.length === 0 ? "+ Pick the first recipe" : "+ Add another recipe"}
        onPress={() => router.push("/(app)/recipes")}
      />

      {basket.items.length > 0 && (
        <View className="mt-[4px] rounded-2xl bg-accent2-200 px-[20px] py-[18px]">
          <View className="flex-row justify-between">
            <Text className="font-fig-bold text-[14px] text-accent2-800">
              {basket.items.length} recipe{basket.items.length === 1 ? "" : "s"} · {batches} batch
              {batches === 1 ? "" : "es"}
            </Text>
            <Text className="font-fig-bold text-[14px] text-accent2-800">
              {formatTotalTime(totalSeconds)}
            </Text>
          </View>
          <PrimaryButton
            title={
              totals.data && totals.data.length > 0
                ? `Shopping list · ${totals.data.length} items`
                : "Generate shopping list"
            }
            onPress={() => router.push("/(app)/basket")}
            className="mt-[14px]"
          />
        </View>
      )}
    </View>
  );
}

function BasketRow({ recipeId, index }: { recipeId: string; index: number }) {
  const recipe = useRecipe(recipeId);
  const basket = useBasket();
  const r = recipe.data;
  const tint = tintFor(r?.categoryId, index);

  return (
    <View className="flex-row items-center gap-[13px] rounded-2xl bg-neutral-100 py-[12px] pl-[12px] pr-[14px]">
      <Avatar initial={initialOf(r?.title ?? "?")} tint={tint} size={52} />
      <View className="min-w-0 flex-1">
        <Text className="font-fig-bold text-[15px] leading-[18px] text-fg" numberOfLines={1}>
          {r?.title ?? "Loading…"}
        </Text>
        <Text className="mt-[3px] font-fig-semi text-[12.5px] text-neutral-600" numberOfLines={1}>
          {r
            ? metaLine([
                formatDuration(r.prepSeconds + r.cookSeconds),
                r.rating > 0 ? `${r.rating}.0 ★` : undefined,
              ])
            : ""}
        </Text>
      </View>
      {/* Stepping below 1 removes the recipe — the basket has no "zero batches" state. */}
      <Stepper
        compact
        min={0}
        value={basket.batchesOf(recipeId)}
        onChange={(n) => basket.setBatches(recipeId, n)}
        label={`batches of ${r?.title ?? "recipe"}`}
      />
    </View>
  );
}

function WeekTab() {
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
    <View className="gap-[9px]">
      {days.map((day) => {
        const dayEntries = entries.filter((e) => e.date === day.iso);
        return (
          <View key={day.iso} className="flex-row items-stretch gap-[13px]">
            <View className="w-[46px] flex-none items-center pt-[13px]">
              <Text className={`font-cap text-[17px] ${day.isToday ? "text-accent-700" : "text-fg"}`}>
                {day.num}
              </Text>
              <Text className="mt-[1px] font-fig-x text-[11px] uppercase tracking-[1px] text-neutral-600">
                {day.day}
              </Text>
            </View>
            <View className="min-w-0 flex-1 gap-[7px]">
              {dayEntries.map((entry) => {
                const recipe = recipes.data?.find((r) => r.id === entry.recipeId);
                const tint = tintFor(recipe?.categoryId);
                return (
                  <Pressable
                    key={entry.id}
                    accessibilityRole="button"
                    accessibilityLabel={`Remove ${recipe?.title ?? "meal"} from ${day.day}`}
                    onPress={() =>
                      Alert.alert("Remove from the week?", recipe?.title ?? "This meal", [
                        { text: "Keep", style: "cancel" },
                        {
                          text: "Remove",
                          style: "destructive",
                          onPress: () => removeEntry.mutate(entry.id),
                        },
                      ])
                    }
                    className="flex-row justify-between gap-[10px] rounded-xl px-[15px] py-[11px]"
                    style={{ backgroundColor: tint.bg }}
                  >
                    <Text
                      className="flex-1 font-fig-bold text-[14px]"
                      style={{ color: tint.fg }}
                      numberOfLines={1}
                    >
                      {recipe?.title ?? "Unknown recipe"}
                    </Text>
                    <Text
                      className="font-fig-bold text-[12.5px] opacity-70"
                      style={{ color: tint.fg }}
                    >
                      {SLOT_LABELS[entry.slot] ?? "Meal"}
                    </Text>
                  </Pressable>
                );
              })}
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={`Plan a meal for ${day.day} ${day.num}`}
                onPress={() => setAddingTo(day.iso)}
                className="flex-row items-center justify-between rounded-xl border-[1.5px] border-dashed border-neutral-400 px-[15px] py-[11px]"
              >
                <Text className="font-fig-bold text-[13.5px] text-neutral-500">
                  {dayEntries.length === 0 ? "Nothing planned" : "Add another"}
                </Text>
                <Icon name="plus" size={15} color={organic.neutral[500]} width={2.4} />
              </Pressable>
            </View>
          </View>
        );
      })}

      <Sheet visible={addingTo !== null} onClose={() => setAddingTo(null)} title="Plan a meal">
        <Kicker className="mb-[10px]">Slot</Kicker>
        <View className="mb-[20px] flex-row flex-wrap gap-[8px]">
          {SLOTS.map((s) => (
            <Chip key={s.label} label={s.label} active={s.value === slot} onPress={() => setSlot(s.value)} />
          ))}
        </View>

        <Kicker className="mb-[10px]">Recipe</Kicker>
        <ScrollView className="max-h-[280px]" showsVerticalScrollIndicator={false}>
          <View className="gap-[8px]">
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
                className="flex-row items-center gap-[12px] rounded-2xl bg-neutral-100 py-[10px] pl-[10px] pr-[14px]"
              >
                <Avatar initial={initialOf(r.title)} tint={tintFor(r.categoryId, index)} size={40} />
                <Text className="flex-1 font-fig-bold text-[14.5px] text-fg" numberOfLines={1}>
                  {r.title}
                </Text>
              </Pressable>
            ))}
            {(recipes.data ?? []).length === 0 && (
              <Text className="font-fig text-[15px] text-neutral-600">
                No recipes to plan yet — add one first.
              </Text>
            )}
          </View>
        </ScrollView>
      </Sheet>
    </View>
  );
}
