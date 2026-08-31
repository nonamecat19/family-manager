import { Text, TextInput, View } from "react-native";

import { Icon, nocturne } from "@/components/nocturne";

export interface AmountRowProps {
  /** The raw text the user is typing — parsed by the screen, never here. */
  value: string;
  onChangeValue: (next: string) => void;
  /** ISO 4217 code of the selected account, drawn beside the figure. */
  currencyCode: string;
  /** Spoken label for the field; the design gives it no visible caption. */
  label: string;
  invalid?: boolean;
}

/**
 * The design's amount line: one underlined right-aligned figure, the currency code in accent
 * beside it, and the calculator glyph. It is a plain `TextInput` with a numeric keyboard
 * rather than an in-app pad — Android's own pad is the pad, and a hand-rolled one would be a
 * second keyboard the user has to learn.
 */
export function AmountRow({ value, onChangeValue, currencyCode, label, invalid = false }: AmountRowProps) {
  return (
    <View className="flex-row items-end justify-center gap-n3">
      <TextInput
        accessibilityLabel={label}
        value={value}
        onChangeText={onChangeValue}
        keyboardType="decimal-pad"
        inputMode="decimal"
        placeholder="0"
        placeholderTextColor={nocturne.neutral[600]}
        selectionColor={nocturne.accent.DEFAULT}
        className="w-[150px] pb-n2 text-right text-[30px] font-medium text-fg"
        style={{
          borderBottomWidth: 1,
          borderBottomColor: invalid ? nocturne.overspend : nocturne.neutral[700],
        }}
      />
      <Text className="pb-n3 text-[15px] font-medium text-accent-400">{currencyCode}</Text>
      <View className="pb-n3">
        <Icon name="calculator" size={19} color={nocturne.neutral[500]} />
      </View>
    </View>
  );
}
