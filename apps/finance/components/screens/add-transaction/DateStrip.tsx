import { Pressable, Text, View } from "react-native";

import { relativeDay, shortDate, type Translate } from "@/components/kit";
import { Icon, organic } from "@fm/ui";

export interface DateStripProps {
  value: string;
  options: readonly string[];
  today: string;
  onChange: (iso: string) => void;
  onOpenCalendar: () => void;
  calendarLabel: string;
  t: Translate;
}

export function DateStrip({
  value,
  options,
  today,
  onChange,
  onOpenCalendar,
  calendarLabel,
  t,
}: DateStripProps) {
  return (
    <View className="flex-row items-center gap-1.4">
      {options.map((iso) => {
        const selected = iso === value;
        const word = relativeDay(t, iso, today);
        return (
          <Pressable
            key={iso}
            accessibilityRole="button"
            accessibilityLabel={`${shortDate(iso)} ${word}`}
            accessibilityState={{ selected }}
            onPress={() => onChange(iso)}
            className={`items-center rounded-md px-2.1 py-1.4 ${selected ? "bg-accent" : ""}`}
          >
            <Text
              className={`text-12 ${selected ? "font-fig-med text-accent-700" : "text-neutral-600"}`}
            >
              {shortDate(iso)}
            </Text>
            <Text className={`text-10 ${selected ? "text-accent-700" : "text-neutral-600"}`}>
              {word}
            </Text>
          </Pressable>
        );
      })}
      <View className="flex-1" />
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={calendarLabel}
        onPress={onOpenCalendar}
        hitSlop={8}
        className="h-9 w-9 items-center justify-center rounded-full"
      >
        <Icon name="calendar-dots" size={20} color={organic.neutral[600]} />
      </Pressable>
    </View>
  );
}
