import { Pressable, Text } from "react-native";
import { Icon, organic, IconName } from "@fm/ui";

export interface DashedActionProps {
  label: string;
  onPress: () => void;
  icon?: IconName;
  className?: string;
}

export function DashedAction({ label, onPress, icon = "folder-plus", className = "" }: DashedActionProps) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      className={`flex-row items-center justify-center gap-1.4 rounded-lg border border-dashed py-2.8 ${className}`}
      style={({ pressed }) => ({ borderColor: organic.neutral[700], opacity: pressed ? 0.8 : 1 })}
    >
      <Icon name={icon} size={17} color={organic.neutral[600]} />
      <Text className="text-13 font-fig-med text-neutral-600">{label}</Text>
    </Pressable>
  );
}
