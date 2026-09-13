import { Text, TextInput, View } from "react-native";

import { Icon, nocturne } from "@/components/nocturne";

export interface AmountRowProps {
  value: string;
  onChangeValue: (next: string) => void;
  currencyCode: string;
  label: string;
  invalid?: boolean;
}

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
