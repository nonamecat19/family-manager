import type { Money } from "@fm/api";
import { Text, View } from "react-native";

import { BudgetBar, Card, Icon, IconCircle, MoneyText, nocturne } from "@/components/nocturne";

/**
 * One row of Home's group list (design canvas, screen 02).
 *
 * It is not `Row` from the kit: a budgeted group draws a progress bar where `Row` draws a
 * subtitle string, and its right-hand caption ("з ₴6,000") turns `overspend` — neither of
 * which `Row` can express. Everything inside is still kit parts, so the row keeps the same
 * surface, tint ramp and money grammar as the rest of the app.
 */
export interface HomeGroupCardProps {
  name: string;
  /** Stored icon key; unknown keys fall back inside `IconCircle`. */
  icon?: string | null;
  /** Series slot from the server, so the row and its donut arc share a colour. */
  colorStep: number;
  amount: Money;
  /** "1 категорія · Сергій" — drawn when the group has no budget bar. */
  meta?: string;
  /** The right-hand caption under the amount: "20%" without a budget, "з ₴16,000" with one. */
  caption?: string;
  /** Present only for a group with a budget; drives the bar and the overspend colour. */
  budget?: { spentMinor: number; limitMinor: number; over: boolean };
  /** Absent for the row that stands for spending with no category: there is nothing to open. */
  onPress?: () => void;
}

export function HomeGroupCard({
  name,
  icon,
  colorStep,
  amount,
  meta,
  caption,
  budget,
  onPress,
}: HomeGroupCardProps) {
  return (
    <Card onPress={onPress} accessibilityLabel={name} padded={false}>
      <View className="flex-row items-center gap-n4 px-n4 py-n4">
        <IconCircle icon={icon} index={colorStep} size={34} />
        <View className="flex-1">
          <Text className="text-[14px] font-medium text-fg" numberOfLines={1}>
            {name}
          </Text>
          {budget ? (
            <BudgetBar
              spentMinor={budget.spentMinor}
              limitMinor={budget.limitMinor}
              height={3}
              valueLabel=""
              className="mt-n2 w-[120px]"
            />
          ) : meta ? (
            <Text className="mt-[2px] text-[10.5px] text-neutral-600" numberOfLines={1}>
              {meta}
            </Text>
          ) : null}
        </View>
        <View className="items-end">
          <MoneyText value={amount} size={14} />
          {caption ? (
            <Text
              className={`mt-[2px] text-[10.5px] ${budget?.over ? "text-overspend" : "text-neutral-600"}`}
            >
              {caption}
            </Text>
          ) : null}
        </View>
        <Icon name="caret-right" size={13} color={nocturne.neutral[700]} />
      </View>
    </Card>
  );
}
