import type { Money } from "@fm/api";
import { Pressable, Text, View } from "react-native";

import { IconCircle, MoneyText } from "@/components/kit";
import { Badge, Divider, Avatar } from "@fm/ui";

export interface TransactionRowProps {
  title: string;
  meta: string;
  icon?: string;
  iconIndex: number;
  memberName: string;
  memberIndex: number;
  amount: Money;
  income?: boolean;
  templateLabel?: string;
  onPress?: () => void;
  onLongPress?: () => void;
  divider?: boolean;
}

export function TransactionRow({
  title,
  meta,
  icon,
  iconIndex,
  memberName,
  memberIndex,
  amount,
  income = false,
  templateLabel,
  onPress,
  onLongPress,
  divider = true,
}: TransactionRowProps) {
  const body = (
    <View className="flex-row items-center gap-2.1 px-2.8 py-2.1">
      <IconCircle icon={icon} index={iconIndex} size={32} />
      <View className="flex-1">
        <Text className="text-[13.5px] text-fg" numberOfLines={1}>
          {title}
        </Text>
        <View className="mt-0.5 flex-row items-center gap-1.4">
          <Avatar name={memberName} index={memberIndex} size={14} />
          <Text className="flex-shrink text-[10.5px] text-neutral-600" numberOfLines={1}>
            {meta}
          </Text>
          {templateLabel ? <Badge label={templateLabel} /> : null}
        </View>
      </View>
      <MoneyText value={amount} size={13.5} weight="medium" tone={income ? "positive" : "default"} />
    </View>
  );

  return (
    <View>
      {onPress || onLongPress ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={title}
          onPress={onPress}
          onLongPress={onLongPress}
          style={({ pressed }) => (pressed ? { opacity: 0.75 } : null)}
        >
          {body}
        </Pressable>
      ) : (
        body
      )}
      {divider ? <Divider /> : null}
    </View>
  );
}
