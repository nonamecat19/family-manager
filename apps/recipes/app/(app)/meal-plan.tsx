import {
  useMealPlan,
  useTotalIngredients,
  useRecipes,
  usePlanMeal,
  useRemoveMealPlanEntry,
} from "@fm/api";
import { MealSlot } from "@fm/sdk/recipes/v1/recipes_pb";
import { Card, EmptyState, ErrorState, Loading } from "@fm/ui";
import { useState } from "react";
import { FlatList, Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

const SLOT_LABELS: Record<number, string> = {
  0: "—",
  1: "Breakfast",
  2: "Lunch",
  3: "Dinner",
  4: "Snack",
  5: "Dessert",
};

function toISO(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export default function MealPlanScreen() {
  const today = new Date();
  const weekStart = new Date(today);
  weekStart.setDate(today.getDate() - today.getDay());
  const weekEnd = new Date(weekStart);
  weekEnd.setDate(weekStart.getDate() + 6);

  const fromISO = toISO(weekStart);
  const toISODate = toISO(weekEnd);

  const mealPlan = useMealPlan(fromISO, toISODate);
  const totals = useTotalIngredients(fromISO, toISODate);
  const recipes = useRecipes();
  const planMeal = usePlanMeal();
  const removeEntry = useRemoveMealPlanEntry();

  const [showTotals, setShowTotals] = useState(false);

  if (mealPlan.isPending) return <Loading label="Loading meal plan…" />;
  if (mealPlan.isError) return <ErrorState message={mealPlan.error.message} onRetry={() => void mealPlan.refetch()} />;

  const entries = mealPlan.data ?? [];
  const days = buildWeek(weekStart);

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="flex-row items-center justify-between p-lg">
        <Text className="text-display font-bold text-fg dark:text-fg-dark">Meal Plan</Text>
        <Pressable
          accessibilityRole="button"
          onPress={() => setShowTotals(!showTotals)}
          className="rounded-lg bg-card dark:bg-card-dark px-md py-sm"
        >
          <Text className="text-body text-fg dark:text-fg-dark">
            {showTotals ? "📅 Calendar" : "🛒 Ingredients"}
          </Text>
        </Pressable>
      </View>

      {showTotals ? (
        totals.isPending ? (
          <Loading />
        ) : totals.isError ? (
          <ErrorState message={totals.error.message} onRetry={() => void totals.refetch()} />
        ) : (
          <FlatList
            data={totals.data ?? []}
            keyExtractor={(t, i) => `${t.name}-${t.unit}-${i}`}
            contentContainerClassName="px-lg pb-2xl gap-xs"
            renderItem={({ item }) => (
              <Card className="flex-row justify-between p-md">
                <Text className="text-body text-fg dark:text-fg-dark">{item.name}</Text>
                <Text className="text-body text-muted dark:text-muted-dark">
                  {item.totalAmount} {item.unit}
                </Text>
              </Card>
            )}
            ListEmptyComponent={
              <EmptyState title="No ingredients to total" hint="Plan some meals first." />
            }
          />
        )
      ) : (
        <ScrollView contentContainerClassName="px-lg pb-2xl gap-md">
          {days.map((day) => {
            const dayEntries = entries.filter((e) => e.date === day.iso);
            return (
              <View key={day.iso} className="gap-xs">
                <Text className="text-title font-semibold text-fg dark:text-fg-dark">
                  {day.label}
                </Text>
                {dayEntries.length > 0 ? (
                  dayEntries.map((entry) => {
                    const recipe = recipes.data?.find((r) => r.id === entry.recipeId);
                    return (
                      <Card key={entry.id} className="flex-row items-center justify-between p-md">
                        <View className="flex-1">
                          <Text className="text-caption text-muted dark:text-muted-dark">
                            {SLOT_LABELS[entry.slot] ?? "Meal"}
                          </Text>
                          <Text className="text-body text-fg dark:text-fg-dark" numberOfLines={1}>
                            {recipe?.title ?? "Unknown recipe"}
                          </Text>
                          {entry.servings > 0 && (
                            <Text className="text-caption text-muted dark:text-muted-dark">
                              {entry.servings} servings
                            </Text>
                          )}
                        </View>
                        <Pressable
                          accessibilityRole="button"
                          accessibilityLabel="Remove from plan"
                          onPress={() => removeEntry.mutate(entry.id)}
                          className="h-8 w-8 items-center justify-center rounded-full bg-card dark:bg-card-dark"
                        >
                          <Text className="text-caption text-error">✕</Text>
                        </Pressable>
                      </Card>
                    );
                  })
                ) : (
                  <Text className="text-caption text-muted dark:text-muted-dark pl-sm">
                    No meals planned
                  </Text>
                )}
              </View>
            );
          })}

          {/* Quick-add: pick a recipe for today's dinner */}
          {recipes.data && recipes.data.length > 0 && (
            <QuickAdd
              recipes={recipes.data.map((r) => ({ id: r.id, title: r.title }))}
              onAdd={(recipeId) =>
                planMeal.mutate({
                  recipeId,
                  date: toISO(today),
                  slot: MealSlot.DINNER,
                  servings: 0,
                })
              }
            />
          )}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

function QuickAdd({
  recipes,
  onAdd,
}: {
  recipes: { id: string; title: string }[];
  onAdd: (recipeId: string) => void;
}) {
  const [selected, setSelected] = useState("");
  return (
    <Card className="gap-sm p-md">
      <Text className="text-title font-semibold text-fg dark:text-fg-dark">Quick add to today</Text>
      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-xs">
        {recipes.map((r) => (
          <Pressable
            key={r.id}
            accessibilityRole="button"
            onPress={() => setSelected(r.id)}
            className={`rounded-full px-md py-sm ${selected === r.id ? "bg-primary" : "bg-card dark:bg-card-dark"}`}
          >
            <Text
              className={`text-caption ${selected === r.id ? "text-primary-fg" : "text-fg dark:text-fg-dark"}`}
            >
              {r.title}
            </Text>
          </Pressable>
        ))}
      </ScrollView>
      {selected !== "" && (
        <Pressable
          accessibilityRole="button"
          onPress={() => {
            onAdd(selected);
            setSelected("");
          }}
          className="items-center rounded-lg bg-primary p-md"
        >
          <Text className="text-body text-primary-fg">Add to dinner</Text>
        </Pressable>
      )}
    </Card>
  );
}

function buildWeek(start: Date): { iso: string; label: string }[] {
  const days: { iso: string; label: string }[] = [];
  const dayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  for (let i = 0; i < 7; i++) {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    days.push({
      iso: toISO(d),
      label: `${dayNames[d.getDay()]} ${d.getDate()}`,
    });
  }
  return days;
}