import { Pressable, Text, View } from "react-native";
import { Icon, Kicker, Avatar, organic, IconName } from "@fm/ui";

export interface PickerFieldProps {
  label: string;
  value: string;
  onPress: () => void;
  icon?: IconName;
  avatar?: { name: string; index: number };
}

export function PickerField({ label, value, onPress, icon, avatar }: PickerFieldProps) {
  return (
    <View className="flex-1">
      <Kicker className="mb-[5.6px]">{label}</Kicker>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${label}: ${value}`}
        onPress={onPress}
        className="flex-row items-center gap-[5.6px] rounded-md bg-surface px-[8.4px] py-[8.4px]"
        style={({ pressed }) => ({
          borderWidth: 1,
          borderColor: organic.neutral[800],
          opacity: pressed ? 0.75 : 1,
        })}
      >
        {avatar ? <Avatar name={avatar.name} index={avatar.index} size={22} /> : null}
        {icon ? <Icon name={icon} size={16} color={organic.accent[600]} /> : null}
        <Text className="flex-1 text-[13px] font-fig-med text-fg" numberOfLines={1}>
          {value}
        </Text>
        <Icon name="caret-down" size={11} color={organic.neutral[600]} />
      </Pressable>
    </View>
  );
}
