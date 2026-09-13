import { Text, View } from "react-native";

import { Card, Icon, nocturne, type IconName } from "@/components/nocturne";

export interface SettingsRowProps {
  icon: IconName;
  label: string;
  meta?: string;
  tone?: "accent" | "neutral";
  onPress: () => void;
}

export function SettingsRow({ icon, label, meta, tone = "neutral", onPress }: SettingsRowProps) {
  return (
    <Card
      onPress={onPress}
      accessibilityLabel={label}
      padded={false}
      className="border border-border px-n5 py-n4"
    >
      <View className="flex-row items-center gap-n4">
        <Icon
          name={icon}
          size={18}
          color={tone === "accent" ? nocturne.accent[400] : nocturne.neutral[500]}
        />
        <View className="flex-1">
          <Text className="text-[13.5px] text-fg" numberOfLines={1}>
            {label}
          </Text>
          {meta ? (
            <Text className="mt-[2px] text-[10.5px] text-neutral-600" numberOfLines={1}>
              {meta}
            </Text>
          ) : null}
        </View>
        <Icon name="caret-right" size={13} color={nocturne.neutral[700]} />
      </View>
    </Card>
  );
}
