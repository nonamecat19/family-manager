import { Text, TextInput, View } from "react-native";
import { Icon, organic } from "@fm/ui";

export interface AmountRowProps {
  value: string;
  onChangeValue: (next: string) => void;
  currencyCode: string;
  label: string;
  invalid?: boolean;
}

export function AmountRow({ value, onChangeValue, currencyCode, label, invalid = false }: AmountRowProps) {
  return (
    <View className="flex-row items-end justify-center gap-[8.4px]">
      <TextInput
        accessibilityLabel={label}
        value={value}
        onChangeText={onChangeValue}
        keyboardType="decimal-pad"
        inputMode="decimal"
        placeholder="0"
        placeholderTextColor={organic.neutral[600]}
        selectionColor={organic.accent.DEFAULT}
        className="w-[150px] pb-[5.6px] text-right text-[30px] font-fig-med text-fg"
        style={{
          borderBottomWidth: 1,
          borderBottomColor: invalid ? organic.danger : organic.neutral[700],
        }}
      />
      <Text className="pb-[8.4px] text-[15px] font-fig-med text-accent-700">{currencyCode}</Text>
      <View className="pb-[8.4px]">
        <Icon name="calculator" size={19} color={organic.neutral[600]} />
      </View>
    </View>
  );
}
