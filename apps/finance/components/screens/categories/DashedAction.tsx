import { Pressable, Text } from "react-native";

import { Icon, nocturne, type IconName } from "@/components/nocturne";

export interface DashedActionProps {
  label: string;
  onPress: () => void;
  icon?: IconName;
  className?: string;
}

/**
 * The dashed "Нова група" footer under the group list. It is deliberately not a `Button`:
 * the design gives the create affordance a dashed outline and a neutral label so it reads as
 * the end of the list rather than the screen's primary action.
 */
export function DashedAction({ label, onPress, icon = "folder-plus", className = "" }: DashedActionProps) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      className={`flex-row items-center justify-center gap-n2 rounded-lg border border-dashed py-n4 ${className}`}
      style={({ pressed }) => ({ borderColor: nocturne.neutral[700], opacity: pressed ? 0.8 : 1 })}
    >
      <Icon name={icon} size={17} color={nocturne.neutral[500]} />
      <Text className="text-[13px] font-medium text-neutral-500">{label}</Text>
    </Pressable>
  );
}
