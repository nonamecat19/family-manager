import type { ReactNode } from "react";
import { Pressable, Text, View } from "react-native";

import { useUiTranslate } from "./labels.tsx";
import { Icon, StarIcon, type IconName } from "./icons.tsx";
import { useTheme } from "./theme.tsx";

export function RoundButton({
  icon,
  label,
  onPress,
  tone = "neutral",
  disabled = false,
  children,
}: {
  icon?: IconName;
  label: string;
  onPress: () => void;
  tone?: "neutral" | "translucent";
  disabled?: boolean;
  children?: ReactNode;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      className={`h-10 w-10 flex-none items-center justify-center rounded-full ${
        tone === "translucent" ? "bg-neutral-100/70" : "bg-neutral-200"
      } ${disabled ? "opacity-50" : ""}`}
    >
      {children ?? (icon ? <Icon name={icon} size={20} /> : null)}
    </Pressable>
  );
}

export function Chip({
  label,
  active,
  onPress,
  tone = "accent",
}: {
  label: string;
  active: boolean;
  onPress: () => void;
  tone?: "accent" | "accent2";
}) {
  const t = useTheme();
  const fill =
    tone === "accent2" ? "border-accent2-500 bg-accent2-200" : "border-accent bg-accent";
  const text = tone === "accent2" ? "text-accent2-800" : "text-white";
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected: active }}
      onPress={onPress}
      className={`flex-none rounded-full border-[1.5px] px-3.75 py-sm ${
        active ? fill : "border-neutral-400 bg-transparent"
      }`}
    >
      <Text
        className={`text-[13.5px] ${active ? text : "text-neutral-700"}`}
        style={{ fontFamily: t.fonts?.bold }}
      >
        {label}
      </Text>
    </Pressable>
  );
}

export function Tag({ label, tone = "accent" }: { label: string; tone?: "accent" | "accent2" }) {
  const t = useTheme();
  return (
    <View
      className={`rounded-full px-2.5 py-0.75 ${
        tone === "accent2" ? "bg-accent2-100" : "bg-accent-100"
      }`}
    >
      <Text
        className={`text-[11px] ${tone === "accent2" ? "text-accent2-800" : "text-accent-800"}`}
        style={{ fontFamily: t.fonts?.semibold }}
      >
        {label}
      </Text>
    </View>
  );
}

export function SegTabs<T extends string>({
  options,
  value,
  onChange,
  labels,
}: {
  options: readonly T[];
  value: T;
  onChange: (value: T) => void;
  labels: Record<T, string>;
}) {
  const t = useTheme();
  return (
    <View className="flex-row gap-1.5 rounded-full bg-neutral-200 p-1.25">
      {options.map((option) => {
        const active = option === value;
        return (
          <Pressable
            key={option}
            accessibilityRole="button"
            accessibilityLabel={labels[option]}
            accessibilityState={{ selected: active }}
            onPress={() => onChange(option)}
            className={`flex-1 items-center rounded-full py-2.25 ${active ? "bg-neutral-100" : ""}`}
          >
            <Text
              className={`text-[14px] ${active ? "text-fg" : "text-neutral-600"}`}
              style={{ fontFamily: t.fonts?.bold }}
            >
              {labels[option]}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

export function PrimaryButton({
  title,
  onPress,
  disabled = false,
  className = "",
}: {
  title: string;
  onPress: () => void;
  disabled?: boolean;
  className?: string;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      className={`items-center rounded-full bg-accent py-3.75 ${disabled ? "opacity-50" : ""} ${className}`}
    >
      <Text className="text-[15.5px] text-white" style={{ fontFamily: t.fonts?.heavy }}>
        {title}
      </Text>
    </Pressable>
  );
}

export function OutlineButton({
  title,
  onPress,
  className = "",
}: {
  title: string;
  onPress: () => void;
  className?: string;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      onPress={onPress}
      className={`flex-none items-center rounded-full border-2 border-accent px-5.5 py-3.25 ${className}`}
    >
      <Text className="text-[15.5px] text-accent-700" style={{ fontFamily: t.fonts?.heavy }}>
        {title}
      </Text>
    </Pressable>
  );
}

export function DashedButton({
  title,
  onPress,
  className = "",
}: {
  title: string;
  onPress: () => void;
  className?: string;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      onPress={onPress}
      className={`items-center rounded-2xl border-2 border-dashed border-neutral-400 py-3.5 ${className}`}
    >
      <Text className="text-[14.5px] text-neutral-700" style={{ fontFamily: t.fonts?.bold }}>
        {title}
      </Text>
    </Pressable>
  );
}

export function Stepper({
  value,
  onChange,
  label,
  min = 1,
  max = 20,
  compact = false,
}: {
  value: number;
  onChange: (value: number) => void;
  label: string;
  min?: number;
  max?: number;
  compact?: boolean;
}) {
  const t = useTheme();
  const tLabel = useUiTranslate();
  const size = compact ? "h-[26px] w-[26px]" : "h-[28px] w-[28px]";
  return (
    <View
      className={`flex-none flex-row items-center rounded-full ${
        compact ? "gap-2.75 bg-bg p-1.25" : "gap-3.5 bg-neutral-100 px-sm py-1.5"
      }`}
    >
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={tLabel("ui.decrease", { label })}
        onPress={() => onChange(Math.max(min, value - 1))}
        className={`${size} items-center justify-center rounded-full bg-neutral-200`}
      >
        <Text className="text-fg" style={{ fontFamily: t.fonts?.heavy }}>
          −
        </Text>
      </Pressable>
      <Text
        className="min-w-[22px] text-center text-[15px] text-fg"
        style={{ fontFamily: t.fonts?.heavy }}
      >
        ×{value}
      </Text>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={tLabel("ui.increase", { label })}
        onPress={() => onChange(Math.min(max, value + 1))}
        className={`${size} items-center justify-center rounded-full bg-accent`}
      >
        <Text className="text-white" style={{ fontFamily: t.fonts?.heavy }}>
          +
        </Text>
      </Pressable>
    </View>
  );
}

export function RatingMark({ rating, size = 13 }: { rating: number; size?: number }) {
  const t = useTheme();
  const tLabel = useUiTranslate();
  if (rating <= 0) return null;
  return (
    <View
      accessible
      accessibilityLabel={tLabel("ui.ratedOutOf5", { rating })}
      className="flex-none flex-row items-center gap-0.75"
    >
      <StarIcon size={size} />
      <Text className="text-[13.5px] text-accent-700" style={{ fontFamily: t.fonts?.heavy }}>
        {rating.toFixed(1)}
      </Text>
    </View>
  );
}

export function StarPicker({
  rating,
  onChange,
  size = 24,
}: {
  rating: number;
  onChange: (rating: number) => void;
  size?: number;
}) {
  const tLabel = useUiTranslate();
  return (
    <View className="flex-row gap-1.5">
      {[1, 2, 3, 4, 5].map((n) => (
        <Pressable
          key={n}
          accessibilityRole="button"
          accessibilityLabel={tLabel("ui.stars", { count: n })}
          accessibilityState={{ selected: n <= rating }}
          onPress={() => onChange(n === rating ? 0 : n)}
        >
          <StarIcon size={size} filled={n <= rating} />
        </Pressable>
      ))}
    </View>
  );
}

export interface IconButtonProps {
  icon: IconName;
  label: string;
  onPress: () => void;
  size?: number;
  color?: string;
  weight?: "regular" | "fill";
  badge?: boolean;
  disabled?: boolean;
}

export function IconButton({
  icon,
  label,
  onPress,
  size = 20,
  color,
  weight = "regular",
  badge = false,
  disabled = false,
}: IconButtonProps) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      hitSlop={8}
      className={`h-9 w-9 flex-none items-center justify-center rounded-xl ${disabled ? "opacity-40" : ""}`}
    >
      <Icon name={icon} size={size} color={color ?? t.text} weight={weight} />
      {badge ? <View className="absolute right-[7px] top-[7px] h-[7px] w-[7px] rounded-full bg-accent" /> : null}
    </Pressable>
  );
}
