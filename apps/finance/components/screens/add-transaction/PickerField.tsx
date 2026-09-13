import { Pressable, Text, View } from "react-native";

import { Icon, Kicker, MemberAvatar, nocturne, type IconName } from "@/components/nocturne";

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
      <Kicker className="mb-n2">{label}</Kicker>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${label}: ${value}`}
        onPress={onPress}
        className="flex-row items-center gap-n2 rounded-md bg-surface px-n3 py-n3"
        style={({ pressed }) => ({
          borderWidth: 1,
          borderColor: nocturne.neutral[800],
          opacity: pressed ? 0.75 : 1,
        })}
      >
        {avatar ? <MemberAvatar name={avatar.name} index={avatar.index} size={22} /> : null}
        {icon ? <Icon name={icon} size={16} color={nocturne.accent[400]} /> : null}
        <Text className="flex-1 text-[13px] font-medium text-fg" numberOfLines={1}>
          {value}
        </Text>
        <Icon name="caret-down" size={11} color={nocturne.neutral[600]} />
      </Pressable>
    </View>
  );
}
