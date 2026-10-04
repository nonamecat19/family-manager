import type { Money } from "@fm/api";
import { Text, View } from "react-native";

import { BudgetBar, Card, IconCircle, MoneyText } from "@/components/kit";
import { Icon, organic } from "@fm/ui";

export interface HomeGroupCardProps {
  name: string;
  icon?: string | null;
  colorStep: number;
  amount: Money;
  meta?: string;
  caption?: string;
  budget?: { spentMinor: number; limitMinor: number; over: boolean };
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
      <View className="flex-row items-center gap-[11.2px] px-[11.2px] py-[11.2px]">
        <IconCircle icon={icon} index={colorStep} size={34} />
        <View className="flex-1">
          <Text className="text-[14px] font-fig-med text-fg" numberOfLines={1}>
            {name}
          </Text>
          {budget ? (
            <BudgetBar
              spentMinor={budget.spentMinor}
              limitMinor={budget.limitMinor}
              height={3}
              valueLabel=""
              className="mt-[5.6px] w-[120px]"
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
              className={`mt-[2px] text-[10.5px] ${budget?.over ? "text-error" : "text-neutral-600"}`}
            >
              {caption}
            </Text>
          ) : null}
        </View>
        <Icon name="caret-right" size={13} color={organic.neutral[700]} />
      </View>
    </Card>
  );
}
