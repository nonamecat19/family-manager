import type { Money } from "@fm/api";
import { Pressable, Text, View } from "react-native";

import { Badge, Divider, IconCircle, MemberAvatar, MoneyText } from "@/components/nocturne";

export interface TransactionRowProps {
  /** The category's name — what the design draws as the row's headline. */
  title: string;
  /** "Mono · Дім" — the account and either the merchant or the category's group. */
  meta: string;
  /** The stored category icon key; unknown keys fall back to the kit's default glyph. */
  icon?: string;
  /** Series slot for the icon's tint, so a category keeps one colour across the feed. */
  iconIndex: number;
  /** Who paid. Every row carries this avatar — it is the point of this screen. */
  memberName: string;
  memberIndex: number;
  amount: Money;
  /** Income is drawn in the positive tone; an expense stays default. */
  income?: boolean;
  /** Draws the "шаблон" badge on a row a quick template wrote. */
  templateLabel?: string;
  onPress?: () => void;
  onLongPress?: () => void;
  divider?: boolean;
}

/**
 * One transaction in the feed.
 *
 * Not the kit's `Row`: the design puts the payer's avatar *inside* the meta line, and `Row`
 * takes its subtitle as a string. Everything else here is kit parts.
 */
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
    <View className="flex-row items-center gap-n3 px-n4 py-n3">
      <IconCircle icon={icon} index={iconIndex} size={32} />
      <View className="flex-1">
        <Text className="text-[13.5px] text-fg" numberOfLines={1}>
          {title}
        </Text>
        <View className="mt-[2px] flex-row items-center gap-n2">
          <MemberAvatar name={memberName} index={memberIndex} size={14} />
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
