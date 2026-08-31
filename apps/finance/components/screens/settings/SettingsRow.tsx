import { Text, View } from "react-native";

import { Card, Icon, nocturne, type IconName } from "@/components/nocturne";

export interface SettingsRowProps {
  icon: IconName;
  label: string;
  /** The small line under the label — "2 учасники · спільний бюджет". */
  meta?: string;
  /**
   * The design draws the first four rows with an accent glyph (the things the household owns)
   * and the last four with a neutral one (device preferences). It is the only difference
   * between the two groups, so it is a tone, not a variant.
   */
  tone?: "accent" | "neutral";
  onPress: () => void;
}

/**
 * Screen 11's settings row: its own rounded surface, not a member of a grouped list — the
 * canvas separates every row with a gap and gives each one its own hairline, so `ListSection`
 * (one clipped surface, internal dividers) would draw a different screen.
 */
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
