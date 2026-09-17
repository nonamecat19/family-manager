import type { Money } from "@fm/api";
import { Icon, iconOr, Kicker, Sheet as UiSheet, type IconName } from "@fm/ui";
import type { ReactNode } from "react";
import { Dimensions, Pressable, ScrollView, Text, View } from "react-native";

import { formatMoney, type MoneyFormatOptions } from "./money.ts";
import { organic, tintFor, type Tint } from "./tokens.ts";

/**
 * The finance-specific chrome: the money read-outs and the few bubbles the shared library does
 * not carry. Everything with a `@fm/ui` counterpart lives there instead.
 */

export interface ListSectionProps {
  title?: string;
  action?: { label: string; onPress: () => void };
  children: ReactNode;
  className?: string;
}

export function ListSection({ title, action, children, className = "" }: ListSectionProps) {
  return (
    <View className={className}>
      {title || action ? (
        <View className="mb-2.1 flex-row items-baseline justify-between">
          {title ? <Kicker>{title}</Kicker> : <View />}
          {action ? (
            <Pressable accessibilityRole="button" accessibilityLabel={action.label} onPress={action.onPress}>
              <Text className="font-fig-bold text-12.5 text-accent-700">{action.label}</Text>
            </Pressable>
          ) : null}
        </View>
      ) : null}
      <View className="overflow-hidden rounded-2xl bg-surface shadow-card">{children}</View>
    </View>
  );
}

export interface RowProps {
  title: string;
  subtitle?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
  trailingSubtitle?: string;
  onPress?: () => void;
  onLongPress?: () => void;
  chevron?: boolean;
  divider?: boolean;
  className?: string;
}

export function Row({
  title,
  subtitle,
  leading,
  trailing,
  trailingSubtitle,
  onPress,
  onLongPress,
  chevron = false,
  divider = true,
  className = "",
}: RowProps) {
  const body = (
    <View className={`flex-row items-center gap-2.8 px-4.2 py-2.8 ${className}`}>
      {leading}
      <View className="flex-1">
        <Text className="font-fig-med text-14.5 text-fg" numberOfLines={1}>
          {title}
        </Text>
        {subtitle ? (
          <Text className="mt-0.5 font-fig text-11.5 text-neutral-600" numberOfLines={1}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing || trailingSubtitle ? (
        <View className="items-end">
          {trailing}
          {trailingSubtitle ? (
            <Text className="mt-0.5 font-fig text-11 text-neutral-600">{trailingSubtitle}</Text>
          ) : null}
        </View>
      ) : null}
      {chevron ? <Icon name="caret-right" size={16} color={organic.neutral[600]} /> : null}
    </View>
  );

  return (
    <View>
      {onPress || onLongPress ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={title}
          onPress={onPress}
          onLongPress={onLongPress}
          style={({ pressed }) => (pressed ? { opacity: 0.75 } : null)}
        >
          {body}
        </Pressable>
      ) : (
        body
      )}
      {divider ? <View className="ml-4.2 h-px w-full bg-divider" /> : null}
    </View>
  );
}

export type MoneyTone = "default" | "muted" | "accent" | "overspend" | "positive" | "onAccent";

export interface MoneyTextProps extends MoneyFormatOptions {
  value: Money;
  size?: number;
  weight?: "regular" | "medium" | "semibold";
  tone?: MoneyTone;
  over?: boolean;
  className?: string;
}

const TONE_CLASS: Record<MoneyTone, string> = {
  default: "text-fg",
  muted: "text-neutral-600",
  accent: "text-accent-700",
  overspend: "text-error",
  positive: "text-accent2-700",
  onAccent: "text-white",
};

const WEIGHT_CLASS = {
  regular: "font-fig",
  medium: "font-fig-med",
  semibold: "font-fig-semi",
} as const;

export function MoneyText({
  value,
  size = 15,
  weight = "medium",
  tone = "default",
  over = false,
  className = "",
  ...format
}: MoneyTextProps) {
  const resolved: MoneyTone = over ? "overspend" : tone;
  return (
    <Text
      className={`${WEIGHT_CLASS[weight]} ${TONE_CLASS[resolved]} ${className}`}
      style={{ fontSize: size, lineHeight: Math.round(size * 1.2) }}
    >
      {formatMoney(value, format)}
    </Text>
  );
}

export interface IconCircleProps {
  icon: string | null | undefined;
  index?: number;
  tint?: Tint;
  size?: number;
}

export function IconCircle({ icon, index = 0, tint, size = 38 }: IconCircleProps) {
  const resolved = tint ?? tintFor(index);
  return (
    <View
      className="items-center justify-center"
      style={{ width: size, height: size, borderRadius: size / 2, backgroundColor: resolved.bg }}
    >
      <Icon name={iconOr(icon)} size={Math.round(size * 0.5)} color={resolved.fg} weight="fill" />
    </View>
  );
}

export interface AmountChipProps {
  label: string;
  amount?: Money;
  icon?: IconName;
  onPress?: () => void;
  onLongPress?: () => void;
  selected?: boolean;
  variant?: "solid" | "outline";
  className?: string;
}

/**
 * A chip that can carry an icon and a money read-out (quick templates, account pickers). The
 * shared `@fm/ui` `Chip` is a plain label toggle and does not reach this far, so finance keeps
 * its own under a distinct name.
 */
export function AmountChip({
  label,
  amount,
  icon,
  onPress,
  onLongPress,
  selected = false,
  variant = "solid",
  className = "",
}: AmountChipProps) {
  const filled = variant !== "outline" && selected;
  const shell = variant === "outline" ? "border border-dashed border-neutral-400" : filled ? "bg-accent" : "bg-surface shadow-card";
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected }}
      onPress={onPress}
      onLongPress={onLongPress}
      className={`flex-none flex-row items-center gap-1.4 rounded-full px-2.8 py-2.1 ${shell} ${className}`}
      style={({ pressed }) => (pressed ? { opacity: 0.75 } : null)}
    >
      {icon ? (
        <Icon name={icon} size={15} color={filled ? organic.accentFg : organic.accent.DEFAULT} />
      ) : null}
      <Text className={`font-fig-med text-12.5 ${filled ? "text-white" : "text-fg"}`}>{label}</Text>
      {amount ? <MoneyText value={amount} size={12.5} tone={filled ? "onAccent" : "accent"} /> : null}
    </Pressable>
  );
}

export function Fab({ label, onPress, icon = "plus" }: { label: string; onPress: () => void; icon?: IconName }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      className="absolute bottom-[26px] right-[18px] h-[56px] w-[56px] items-center justify-center rounded-full bg-accent shadow-fab"
      style={({ pressed }) => ({ opacity: pressed ? 0.85 : 1 })}
    >
      <Icon name={icon} size={24} color={organic.accentFg} />
    </Pressable>
  );
}

export function Stat({ label, value, tone = "default" }: { label: string; value: string; tone?: MoneyTone }) {
  return (
    <View className="flex-1">
      <Text className="font-fig-bold text-10.5 uppercase tracking-[0.8px] text-neutral-600">{label}</Text>
      <Text className={`mt-0.75 font-fig-med text-15 ${TONE_CLASS[tone]}`}>{value}</Text>
    </View>
  );
}

export interface CardProps {
  children: ReactNode;
  onPress?: () => void;
  onLongPress?: () => void;
  accessibilityLabel?: string;
  padded?: boolean;
  className?: string;
}

export function Card({
  children,
  onPress,
  onLongPress,
  accessibilityLabel,
  padded = true,
  className = "",
}: CardProps) {
  const base = `rounded-xl border border-divider bg-surface shadow-card ${padded ? "p-4.2" : ""} ${className}`;
  if (!onPress && !onLongPress) return <View className={base}>{children}</View>;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      onPress={onPress}
      onLongPress={onLongPress}
      className={base}
      style={({ pressed }) => (pressed ? { opacity: 0.82 } : null)}
    >
      {children}
    </Pressable>
  );
}

export interface ScrollSheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  scroll?: boolean;
}

/**
 * The shared `@fm/ui` `Sheet` renders its children as-is; several finance sheets (account and
 * category pickers, filters) are long lists, so this wraps it with the scrolling body those need.
 */
export function ScrollSheet({ visible, onClose, title, children, scroll = true }: ScrollSheetProps) {
  const maxScroll = Math.round(Dimensions.get("window").height * 0.6);
  return (
    <UiSheet visible={visible} onClose={onClose} title={title}>
      {scroll ? (
        <ScrollView
          style={{ maxHeight: maxScroll }}
          contentContainerStyle={{ paddingBottom: 8 }}
          keyboardShouldPersistTaps="handled"
        >
          {children}
        </ScrollView>
      ) : (
        children
      )}
    </UiSheet>
  );
}
