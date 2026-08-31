import { Pressable, Text, View } from "react-native";

import { Icon, nocturne, relativeDay, shortDate, type Translate } from "@/components/nocturne";

export interface DateStripProps {
  /** The chosen day, `YYYY-MM-DD`. */
  value: string;
  /** The three offered days, newest first — today, yesterday, the day before. */
  options: readonly string[];
  today: string;
  onChange: (iso: string) => void;
  /** Opens the fuller day list. */
  onOpenCalendar: () => void;
  calendarLabel: string;
  t: Translate;
}

/**
 * "8/30 сьогодні · 8/29 вчора · 8/28 2 дні", then the calendar glyph. Three chips because
 * that is what a transaction typed after the fact almost always is; anything older goes
 * through the calendar.
 */
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
    <View className="flex-row items-center gap-n2">
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
            className={`items-center rounded-md px-n3 py-n2 ${selected ? "bg-accent-900" : ""}`}
          >
            <Text
              className={`text-[12px] ${selected ? "font-medium text-accent-200" : "text-neutral-400"}`}
            >
              {shortDate(iso)}
            </Text>
            <Text className={`text-[10px] ${selected ? "text-accent-200" : "text-neutral-600"}`}>
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
        <Icon name="calendar-dots" size={20} color={nocturne.neutral[500]} />
      </Pressable>
    </View>
  );
}
