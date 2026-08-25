import type { ReactNode } from "react";
import {
  Modal,
  Pressable,
  Text,
  TextInput,
  View,
  type TextInputProps,
  type ViewProps,
} from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";

import { formatMacro } from "./format.ts";
import { Icon, StarIcon, type IconName } from "./icons.tsx";
import { organic, type Tint } from "./tokens.ts";

/** Every screen sits on the same warm ground; no screen paints its own background. */
export function Screen({
  children,
  edges,
  className = "",
}: {
  children: ReactNode;
  edges?: readonly Edge[];
  className?: string;
}) {
  return (
    <SafeAreaView className={`flex-1 bg-bg ${className}`} edges={edges}>
      {children}
    </SafeAreaView>
  );
}

/** The small uppercase label above a group. Organic's `h6`. */
export function Kicker({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <Text className={`font-fig-bold text-[12px] uppercase tracking-[1.2px] text-neutral-600 ${className}`}>
      {children}
    </Text>
  );
}

/** Display type is always Caprasimo — never a bold body face. */
export function Display({
  children,
  size = 28,
  className = "",
}: {
  children: ReactNode;
  size?: number;
  className?: string;
}) {
  return (
    <Text
      className={`font-cap text-fg ${className}`}
      style={{ fontSize: size, lineHeight: size * 1.12 }}
    >
      {children}
    </Text>
  );
}

/** A 40px circular control — back, close, favourite. */
export function RoundButton({
  icon,
  label,
  onPress,
  tone = "neutral",
  children,
}: {
  icon?: IconName;
  label: string;
  onPress: () => void;
  tone?: "neutral" | "translucent";
  children?: ReactNode;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      className={`h-10 w-10 flex-none items-center justify-center rounded-full ${
        tone === "translucent" ? "bg-neutral-100/70" : "bg-neutral-200"
      }`}
    >
      {children ?? (icon ? <Icon name={icon} size={20} /> : null)}
    </Pressable>
  );
}

/**
 * A filter/facet pill. Inactive is an outline so a row of them reads as one control group;
 * active fills with the accent (or the olive secondary for the softer facets).
 */
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
  const fill =
    tone === "accent2"
      ? "border-accent2-500 bg-accent2-200"
      : "border-accent bg-accent";
  const text = tone === "accent2" ? "text-accent2-800" : "text-white";
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected: active }}
      onPress={onPress}
      className={`flex-none rounded-full border-[1.5px] px-[15px] py-[8px] ${
        active ? fill : "border-neutral-400 bg-transparent"
      }`}
    >
      <Text
        className={`font-fig-bold text-[13.5px] ${active ? text : "text-neutral-700"}`}
      >
        {label}
      </Text>
    </Pressable>
  );
}

/** A static category/subcategory label. Not pressable — that's a Chip. */
export function Tag({ label, tone = "accent" }: { label: string; tone?: "accent" | "accent2" }) {
  return (
    <View
      className={`rounded-full px-[10px] py-[3px] ${tone === "accent2" ? "bg-accent2-100" : "bg-accent-100"}`}
    >
      <Text
        className={`font-fig-semi text-[11px] ${tone === "accent2" ? "text-accent2-800" : "text-accent-800"}`}
      >
        {label}
      </Text>
    </View>
  );
}

/** The inset segmented control used for detail tabs and the plan's Basket/Week switch. */
export function SegTabs<T extends string>({
  options,
  value,
  onChange,
}: {
  options: readonly T[];
  value: T;
  onChange: (value: T) => void;
}) {
  return (
    <View className="flex-row gap-[6px] rounded-full bg-neutral-200 p-[5px]">
      {options.map((option) => {
        const active = option === value;
        return (
          <Pressable
            key={option}
            accessibilityRole="button"
            accessibilityLabel={option}
            accessibilityState={{ selected: active }}
            onPress={() => onChange(option)}
            className={`flex-1 items-center rounded-full py-[9px] ${active ? "bg-neutral-100" : ""}`}
          >
            <Text className={`font-fig-bold text-[14px] ${active ? "text-fg" : "text-neutral-600"}`}>
              {option}
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
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      className={`items-center rounded-full bg-accent py-[15px] ${disabled ? "opacity-50" : ""} ${className}`}
    >
      <Text className="font-fig-x text-[15.5px] text-white">{title}</Text>
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
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      onPress={onPress}
      className={`flex-none items-center rounded-full border-2 border-accent px-[22px] py-[13px] ${className}`}
    >
      <Text className="font-fig-x text-[15.5px] text-accent-700">{title}</Text>
    </Pressable>
  );
}

/** The "+ Add another…" affordance: a dashed outline, never a filled button. */
export function DashedButton({
  title,
  onPress,
  className = "",
}: {
  title: string;
  onPress: () => void;
  className?: string;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      onPress={onPress}
      className={`items-center rounded-2xl border-2 border-dashed border-neutral-400 py-[14px] ${className}`}
    >
      <Text className="font-fig-bold text-[14.5px] text-neutral-700">{title}</Text>
    </Pressable>
  );
}

/** − ×n + — batches on the detail screen, servings in the plan. */
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
  const size = compact ? "h-[26px] w-[26px]" : "h-[28px] w-[28px]";
  return (
    <View
      className={`flex-none flex-row items-center rounded-full ${
        compact ? "gap-[11px] bg-bg p-[5px]" : "gap-[14px] bg-neutral-100 px-[8px] py-[6px]"
      }`}
    >
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`Decrease ${label}`}
        onPress={() => onChange(Math.max(min, value - 1))}
        className={`${size} items-center justify-center rounded-full bg-neutral-200`}
      >
        <Text className="font-fig-x text-fg">−</Text>
      </Pressable>
      <Text className="min-w-[22px] text-center font-fig-x text-[15px] text-fg">×{value}</Text>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`Increase ${label}`}
        onPress={() => onChange(Math.min(max, value + 1))}
        className={`${size} items-center justify-center rounded-full bg-accent`}
      >
        <Text className="font-fig-x text-white">+</Text>
      </Pressable>
    </View>
  );
}

/** The circle a recipe or person is represented by when there is no photo. */
export function Avatar({
  initial,
  tint,
  size = 52,
}: {
  initial: string;
  tint: Tint;
  size?: number;
}) {
  return (
    <View
      className="flex-none items-center justify-center rounded-full"
      style={{ width: size, height: size, backgroundColor: tint.bg }}
    >
      <Text className="font-cap" style={{ fontSize: size * 0.35, color: tint.fg }}>
        {initial}
      </Text>
    </View>
  );
}

/** Read-only rating: the star plus the number, as the cards show it. */
export function RatingMark({ rating, size = 13 }: { rating: number; size?: number }) {
  if (rating <= 0) return null;
  return (
    <View
      accessible
      accessibilityLabel={`Rated ${rating} out of 5`}
      className="flex-none flex-row items-center gap-[3px]"
    >
      <StarIcon size={size} />
      <Text className="font-fig-x text-[13.5px] text-accent-700">{rating.toFixed(1)}</Text>
    </View>
  );
}

/**
 * Tapping the already-selected star clears the rating — the 0-means-unrated convention the
 * service uses needs a way back, and a separate "clear" control for one field is worse.
 */
export function StarPicker({
  rating,
  onChange,
  size = 24,
}: {
  rating: number;
  onChange: (rating: number) => void;
  size?: number;
}) {
  return (
    <View className="flex-row gap-[6px]">
      {[1, 2, 3, 4, 5].map((n) => (
        <Pressable
          key={n}
          accessibilityRole="button"
          accessibilityLabel={`${n} star${n === 1 ? "" : "s"}`}
          accessibilityState={{ selected: n <= rating }}
          onPress={() => onChange(n === rating ? 0 : n)}
        >
          <StarIcon size={size} filled={n <= rating} />
        </Pressable>
      ))}
    </View>
  );
}

/** A bottom sheet with a scrim. Used for the list screen's filters. */
export function Sheet({
  visible,
  onClose,
  title,
  children,
}: {
  visible: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}) {
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <View className="flex-1 justify-end">
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Close"
          onPress={onClose}
          className="absolute inset-0"
          style={{ backgroundColor: "rgba(32,30,29,0.4)" }}
        />
        <View className="rounded-t-3xl bg-bg px-[22px] pb-[34px] pt-[20px]">
          <View className="mb-[18px] h-[5px] w-[44px] self-center rounded-full bg-neutral-400" />
          <Display size={22} className="mb-[16px]">
            {title}
          </Display>
          {children}
        </View>
      </View>
    </Modal>
  );
}

/**
 * A labelled text input. Organic's `.field` + `.input`: pill-shaped, on the light neutral,
 * with the label above rather than a placeholder standing in for it.
 */
export function Field({
  label,
  value,
  onChangeText,
  placeholder,
  multiline = false,
  keyboardType,
  autoCapitalize,
  secureTextEntry,
  autoComplete,
  className = "",
  error,
}: {
  label: string;
  value: string;
  onChangeText: (value: string) => void;
  placeholder?: string;
  multiline?: boolean;
  keyboardType?: TextInputProps["keyboardType"];
  autoCapitalize?: TextInputProps["autoCapitalize"];
  secureTextEntry?: boolean;
  autoComplete?: TextInputProps["autoComplete"];
  className?: string;
  error?: string;
}) {
  return (
    <View className={className}>
      <Text className="mb-[5px] font-fig-semi text-[12px] text-neutral-700">{label}</Text>
      <TextInput
        accessibilityLabel={label}
        value={value}
        onChangeText={onChangeText}
        placeholder={placeholder}
        placeholderTextColor={organic.neutral[500]}
        multiline={multiline}
        keyboardType={keyboardType}
        autoCapitalize={autoCapitalize}
        secureTextEntry={secureTextEntry}
        autoComplete={autoComplete}
        className={`rounded-full border bg-neutral-100 px-[14px] py-[11px] font-fig text-[14px] text-fg ${
          multiline ? "min-h-[86px] rounded-2xl" : ""
        } ${error ? "border-[#a5341f]" : "border-divider"}`}
        style={multiline ? { textAlignVertical: "top" } : undefined}
      />
      {error ? (
        <Text className="mt-[5px] font-fig-semi text-[12px]" style={{ color: organic.danger }}>
          {error}
        </Text>
      ) : null}
    </View>
  );
}

/** A white-ish panel: list groups, cards, rows. */
export function Panel({ className = "", ...props }: ViewProps & { className?: string }) {
  return <View className={`rounded-2xl bg-neutral-100 ${className}`} {...props} />;
}

/** Per-serving macros, as a four-cell strip. Renders nothing when the recipe carries no
 * figures — every field is 0 for a recipe nobody recorded them for, and four zeroes read as
 * a claim rather than as an absence. */
export function NutritionStrip({
  kcal,
  proteinG,
  fatG,
  carbsG,
  className = "",
}: {
  kcal: number;
  proteinG: number;
  fatG: number;
  carbsG: number;
  className?: string;
}) {
  if (kcal <= 0 && proteinG <= 0 && fatG <= 0 && carbsG <= 0) return null;
  const cells: { label: string; value: string }[] = [
    { label: "kcal", value: String(Math.round(kcal)) },
    { label: "protein", value: `${formatMacro(proteinG)} g` },
    { label: "fat", value: `${formatMacro(fatG)} g` },
    { label: "carbs", value: `${formatMacro(carbsG)} g` },
  ];
  return (
    <Panel className={`flex-row px-[6px] py-[12px] ${className}`}>
      {cells.map((c) => (
        <View key={c.label} className="flex-1 items-center">
          <Text className="font-cap text-[17px] text-accent-800">{c.value}</Text>
          <Text className="mt-[2px] font-fig text-[11px] uppercase tracking-[0.7px] text-neutral-600">
            {c.label}
          </Text>
        </View>
      ))}
    </Panel>
  );
}

export { organic };
