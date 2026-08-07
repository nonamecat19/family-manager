import { formatRange, periodRange, shiftPeriod, type DateRange, type PeriodKind } from "@fm/api";
import { Pressable, Text, View } from "react-native";

export type SteppablePeriod = Exclude<PeriodKind, "custom" | "all">;

export interface PeriodHeaderProps {
  kind: SteppablePeriod;
  range: DateRange;
  onChange: (kind: SteppablePeriod, range: DateRange) => void;
}

const KINDS: readonly SteppablePeriod[] = ["day", "week", "month", "year"];

/** The period selector that sits above every report: ◀ March 2026 ▶ plus the granularity. */
export function PeriodHeader({ kind, range, onChange }: PeriodHeaderProps) {
  const step = (by: number) => onChange(kind, shiftPeriod(range, kind, by));

  return (
    <View className="gap-sm">
      <View className="flex-row items-center justify-between">
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Previous period"
          hitSlop={12}
          onPress={() => step(-1)}
        >
          <Text className="text-title text-primary">‹</Text>
        </Pressable>

        <Text className="text-title font-semibold text-fg dark:text-fg-dark">
          {formatRange(range)}
        </Text>

        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Next period"
          hitSlop={12}
          onPress={() => step(1)}
        >
          <Text className="text-title text-primary">›</Text>
        </Pressable>
      </View>

      <View className="flex-row gap-xs">
        {KINDS.map((k) => (
          <Pressable
            key={k}
            accessibilityRole="button"
            accessibilityState={{ selected: k === kind }}
            // Switching granularity re-anchors on today rather than trying to reinterpret the
            // current range, which is what a user pressing "year" expects.
            onPress={() => onChange(k, periodRange(k))}
            className={`flex-1 rounded-md py-xs ${
              k === kind ? "bg-primary" : "bg-transparent border border-border dark:border-border-dark"
            }`}
          >
            <Text
              className={`text-center text-caption capitalize ${
                k === kind ? "text-primary-fg" : "text-muted dark:text-muted-dark"
              }`}
            >
              {k}
            </Text>
          </Pressable>
        ))}
      </View>
    </View>
  );
}
