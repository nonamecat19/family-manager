import { useMealPlan, useSumIngredients, useTotalIngredients } from "@fm/api";
import type { IngredientTotal } from "@fm/sdk/recipes/v1/recipes_pb";
import { useRouter } from "expo-router";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useBasket } from "../../components/basket.tsx";
import { useI18n } from "../../components/i18n/index.tsx";
import { groupByAisle } from "../../components/organic/aisles.ts";
import { CheckIcon } from "../../components/organic/icons.tsx";
import { weekRange } from "../../components/organic/week.ts";
import { organic } from "../../components/organic/tokens.ts";
import { DashedButton, Display, Screen } from "../../components/organic/ui.tsx";

/**
 * The shopping list. It is a view of a total, not a thing of its own: whatever is in the
 * plan's basket gets summed by the service, grouped into aisles here, and ticked off
 * locally. Nothing is written back — the family's phone is the trolley.
 */
export default function ShoppingListScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const basket = useBasket();
  const { from, to } = weekRange(new Date());

  const basketTotals = useSumIngredients(basket.items);
  // With an empty basket the list falls back to the week that is already planned, so the
  // tab is useful on a Saturday morning without re-picking every recipe.
  const weekPlan = useMealPlan(from, to);
  const weekTotals = useTotalIngredients(from, to);

  const usingBasket = basket.items.length > 0;
  const totals: IngredientTotal[] = (usingBasket ? basketTotals.data : weekTotals.data) ?? [];
  const sourceCount = usingBasket
    ? basket.items.length
    : new Set((weekPlan.data ?? []).map((e) => e.recipeId)).size;

  const groups = groupByAisle(totals);
  const bought = totals.filter((t) => basket.checked[keyOf(t)]).length;

  return (
    <Screen>
      <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="gap-[18px] px-[22px] pb-[28px] pt-[8px]">
        <View>
          <Display size={28}>{t("shoppingList.title")}</Display>
          <Text className="mt-[5px] font-fig-bold text-[13px] text-neutral-600">
            {totals.length === 0
              ? t("shoppingList.nothingToBuyYet")
              : t("shoppingList.fromRecipesInBasket", {
                  recipes: t("plurals.recipesCount", { count: sourceCount }),
                  bought,
                  total: totals.length,
                })}
          </Text>
          {!usingBasket && totals.length > 0 && (
            <Text className="mt-[3px] font-fig-semi text-[12.5px] text-neutral-500">
              {t("shoppingList.totalledFromWeek")}
            </Text>
          )}
        </View>

        {groups.map((group) => (
          <View key={group.aisle.id}>
            <View className="mb-[9px] flex-row items-center gap-[9px]">
              <View
                className="h-[9px] w-[9px] rounded-full"
                style={{ backgroundColor: group.aisle.dot }}
              />
              <Text className="font-fig-x text-[12px] uppercase tracking-[1.2px] text-neutral-700">
                {t(group.aisle.nameKey)}
              </Text>
            </View>
            <View className="rounded-2xl bg-neutral-100 px-[16px] py-[4px]">
              {group.items.map((item, i) => {
                const key = keyOf(item);
                const on = basket.checked[key] === true;
                return (
                  <Pressable
                    key={key}
                    accessibilityRole="checkbox"
                    accessibilityLabel={item.name}
                    accessibilityState={{ checked: on }}
                    onPress={() => basket.toggleChecked(key)}
                    className={`flex-row items-center gap-[13px] py-[12px] ${
                      i === group.items.length - 1 ? "" : "border-b border-divider"
                    }`}
                  >
                    <View
                      className="h-[23px] w-[23px] flex-none items-center justify-center rounded-full border-2"
                      style={{
                        backgroundColor: on ? organic.accent.DEFAULT : "transparent",
                        borderColor: on ? organic.accent.DEFAULT : organic.neutral[400],
                      }}
                    >
                      {on && <CheckIcon />}
                    </View>
                    <Text
                      className={`flex-1 font-fig-semi text-[15.5px] text-fg ${on ? "opacity-45" : ""}`}
                      style={on ? { textDecorationLine: "line-through" } : undefined}
                    >
                      {item.name}
                    </Text>
                    <Text className={`font-fig-x text-[14px] text-accent-700 ${on ? "opacity-45" : ""}`}>
                      {`${item.totalAmount} ${item.unit}`.trim()}
                    </Text>
                  </Pressable>
                );
              })}
            </View>
          </View>
        ))}

        {totals.length === 0 ? (
          <View className="gap-[12px] pt-[8px]">
            <Text className="font-fig text-[15px] leading-[22px] text-neutral-600">
              {t("shoppingList.emptyBody")}
            </Text>
            <DashedButton title={t("shoppingList.openThePlan")} onPress={() => router.push("/(app)/meal-plan")} />
          </View>
        ) : (
          bought > 0 && (
            <DashedButton title={t("shoppingList.untickEverything")} onPress={() => basket.clearChecked()} />
          )
        )}
      </ScrollView>
    </Screen>
  );
}

/** Totals are summed by name+unit on the server, so that pair is the row's identity. */
function keyOf(total: IngredientTotal): string {
  return `${total.name}|${total.unit}`;
}
