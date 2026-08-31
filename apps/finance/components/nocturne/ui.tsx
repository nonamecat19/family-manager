import type { Money } from "@fm/api";
import type { ReactNode } from "react";
import {
  Modal,
  Pressable,
  ScrollView,
  Switch,
  Text,
  TextInput,
  View,
  type TextInputProps,
} from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import Svg, { Defs, LinearGradient, Rect, Stop } from "react-native-svg";

import { Icon, iconOr, type IconName } from "./icons.tsx";
import { formatMoney, type MoneyFormatOptions } from "./money.ts";
import { initialOf, memberColor, nocturne, tintFor, type Tint } from "./tokens.ts";

/**
 * The Nocturne kit. Every repeated shape in the eleven designed screens lives here, so a
 * screen file is composition and data and nothing else — no raw hex, no raw type sizes, no
 * second version of the same row.
 *
 * Nocturne is DARK ONLY: there is no light pass of these screens, and nothing here reads
 * `dark:` classes or the colour scheme.
 */

// ---- screen scaffolding ---------------------------------------------------------------------

/** Every screen sits on the same ground; no screen paints its own background. */
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

/** The header's top-to-bottom gradient, drawn in SVG rather than pulled in as
 * expo-linear-gradient — a new native module for two stops is not a row this repo's stack
 * table should grow. */
function HeaderGradient() {
  return (
    <View pointerEvents="none" style={{ position: "absolute", top: 0, left: 0, right: 0, bottom: 0 }}>
      <Svg width="100%" height="100%">
        <Defs>
          <LinearGradient id="nocturne-header" x1="0" y1="0" x2="0" y2="1">
            <Stop offset="0" stopColor={nocturne.headerGradient[0]} />
            <Stop offset="1" stopColor={nocturne.headerGradient[1]} />
          </LinearGradient>
        </Defs>
        <Rect x="0" y="0" width="100%" height="100%" fill="url(#nocturne-header)" />
      </Svg>
    </View>
  );
}

export interface HeaderAction {
  icon: IconName;
  /** Spoken label — required, because every one of these is an icon with no text beside it. */
  label: string;
  onPress: () => void;
  /** Draws the accent dot the design puts on an action with something waiting behind it. */
  badge?: boolean;
}

export interface ScreenHeaderProps {
  title: string;
  subtitle?: string;
  /** The left-hand icon: the drawer button on a root screen, back on a pushed one. */
  leading?: HeaderAction;
  /** One or more right-hand icons, drawn in the order given. */
  trailing?: HeaderAction | HeaderAction[];
  /** Paints the accent gradient behind the header (Home and the detail screens); off gives a
   * flat `bg` header (the list screens). */
  gradient?: boolean;
  /** Anything that belongs inside the header block under the title — a segmented control, a
   * scope switcher, a balance. */
  children?: ReactNode;
  className?: string;
}

export function ScreenHeader({
  title,
  subtitle,
  leading,
  trailing,
  gradient = false,
  children,
  className = "",
}: ScreenHeaderProps) {
  const actions = trailing == null ? [] : Array.isArray(trailing) ? trailing : [trailing];
  return (
    <View className={`px-n5 pb-n4 pt-n3 ${className}`}>
      {gradient ? <HeaderGradient /> : null}
      <View className="flex-row items-center gap-n3">
        {leading ? <IconButton {...leading} /> : null}
        <View className="flex-1">
          <Text className="text-[19px] font-medium leading-[24px] text-fg">{title}</Text>
          {subtitle ? <Text className="mt-[2px] text-[12px] text-neutral-500">{subtitle}</Text> : null}
        </View>
        {actions.map((action) => (
          <IconButton key={action.label} {...action} />
        ))}
      </View>
      {children}
    </View>
  );
}

/** A 36px tappable icon. The one place a bare icon is allowed to be a control. */
export function IconButton({ icon, label, onPress, badge, size = 20, color = nocturne.text }: HeaderAction & {
  size?: number;
  color?: string;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      hitSlop={8}
      className="h-9 w-9 flex-none items-center justify-center rounded-full"
    >
      <Icon name={icon} size={size} color={color} />
      {badge ? (
        <View className="absolute right-[7px] top-[7px] h-[7px] w-[7px] rounded-full bg-accent" />
      ) : null}
    </Pressable>
  );
}

/** The small uppercase label above a group — Nocturne's section eyebrow. */
export function Kicker({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <Text
      className={`text-[10.5px] font-medium uppercase text-neutral-600 ${className}`}
      style={{ letterSpacing: 1.05 }}
    >
      {children}
    </Text>
  );
}

/** The hairline the design uses between rows and above a footer. */
export function Divider({ className = "" }: { className?: string }) {
  return <View className={`h-[1px] w-full ${className}`} style={{ backgroundColor: nocturne.divider }} />;
}

/** Android's gesture pill. Drawn because the design draws it; it is decoration, not a control. */
export function BottomIndicator() {
  return (
    <View className="items-center py-n2" pointerEvents="none">
      <View className="h-[4px] w-[128px] rounded-full" style={{ backgroundColor: nocturne.neutral[700] }} />
    </View>
  );
}

// ---- surfaces --------------------------------------------------------------------------------

export interface CardProps {
  children: ReactNode;
  /** Makes the whole card a control. Adds the role and the pressed state; without it the card
   * is inert and must not be wrapped in a Pressable by the caller either. */
  onPress?: () => void;
  onLongPress?: () => void;
  accessibilityLabel?: string;
  /** Off for a card that lays out its own padding (a chart, a full-bleed list). */
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
  const base = `rounded-lg bg-surface ${padded ? "p-n5" : ""} ${className}`;
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

export interface ListSectionProps {
  /** Rendered as a Kicker above the group. Omit for an unlabelled group. */
  title?: string;
  /** The right-hand affordance on the section's own header row — "Додати", "Історія". */
  action?: { label: string; onPress: () => void };
  children: ReactNode;
  className?: string;
}

/** A titled group of rows on one surface. Rows separate themselves; this owns the heading,
 * the rounding and the clip. */
export function ListSection({ title, action, children, className = "" }: ListSectionProps) {
  return (
    <View className={className}>
      {title || action ? (
        <View className="mb-n3 flex-row items-baseline justify-between">
          {title ? <Kicker>{title}</Kicker> : <View />}
          {action ? (
            <Pressable accessibilityRole="button" accessibilityLabel={action.label} onPress={action.onPress}>
              <Text className="text-[12.5px] font-medium text-accent-400">{action.label}</Text>
            </Pressable>
          ) : null}
        </View>
      ) : null}
      <View className="overflow-hidden rounded-lg bg-surface">{children}</View>
    </View>
  );
}

export interface RowProps {
  title: string;
  subtitle?: string;
  /** Usually an `IconCircle` or a `MemberAvatar`. */
  leading?: ReactNode;
  /** Usually a `MoneyText`. Drawn right-aligned. */
  trailing?: ReactNode;
  /** Small right-aligned line under `trailing` — "з ₴16,000", "65%". */
  trailingSubtitle?: string;
  onPress?: () => void;
  onLongPress?: () => void;
  /** Draws the caret the design puts on a row that opens something. */
  chevron?: boolean;
  /** Hairline under the row. Off on the last row of a section. */
  divider?: boolean;
  className?: string;
}

/** The list row every screen is built out of. */
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
    <View className={`flex-row items-center gap-n4 px-n5 py-n4 ${className}`}>
      {leading}
      <View className="flex-1">
        <Text className="text-[14.5px] font-medium text-fg" numberOfLines={1}>
          {title}
        </Text>
        {subtitle ? (
          <Text className="mt-[2px] text-[11.5px] text-neutral-500" numberOfLines={1}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing || trailingSubtitle ? (
        <View className="items-end">
          {trailing}
          {trailingSubtitle ? (
            <Text className="mt-[2px] text-[11px] text-neutral-500">{trailingSubtitle}</Text>
          ) : null}
        </View>
      ) : null}
      {chevron ? <Icon name="caret-right" size={16} color={nocturne.neutral[600]} /> : null}
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
      {divider ? <Divider className="ml-n5" /> : null}
    </View>
  );
}

// ---- money -----------------------------------------------------------------------------------

export type MoneyTone = "default" | "muted" | "accent" | "overspend" | "positive";

export interface MoneyTextProps extends MoneyFormatOptions {
  value: Money;
  /** Font size in px. The design runs 11 (widget) → 34 (the period total on Home). */
  size?: number;
  weight?: "regular" | "medium" | "semibold";
  tone?: MoneyTone;
  /** Shorthand for `tone="overspend"` — what a budget row passes straight from `budgetState`. */
  over?: boolean;
  className?: string;
}

const TONE_CLASS: Record<MoneyTone, string> = {
  default: "text-fg",
  muted: "text-neutral-500",
  accent: "text-accent-300",
  overspend: "text-overspend",
  positive: "text-accent-300",
};

const WEIGHT_CLASS = {
  regular: "font-normal",
  medium: "font-medium",
  semibold: "font-semibold",
} as const;

/**
 * An amount. The ONLY place a figure is drawn, so the minus sign, the grouping and the
 * overspend colour are the same on all eleven screens.
 */
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

// ---- people and categories --------------------------------------------------------------------

export interface MemberAvatarProps {
  /** The member's display name; the avatar shows its first letter. */
  name: string;
  /** Household order, stable — decides the colour. The first two land on the design's own
   * two member colours. */
  index?: number;
  size?: number;
  /** Overrides the index-derived colour, for a member whose colour is stored. */
  color?: string;
  /** Draws the accent ring the design puts on the selected member pill. */
  selected?: boolean;
}

export function MemberAvatar({ name, index = 0, size = 26, color, selected = false }: MemberAvatarProps) {
  const fill = color ?? memberColor(index);
  return (
    <View
      accessibilityRole="image"
      accessibilityLabel={name}
      className="items-center justify-center"
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        backgroundColor: fill,
        borderWidth: selected ? 1.5 : 0,
        borderColor: nocturne.accent[200],
      }}
    >
      <Text style={{ fontSize: size * 0.44, fontWeight: "600", color: nocturne.bg }}>
        {initialOf(name)}
      </Text>
    </View>
  );
}

export interface IconCircleProps {
  /** A stored icon key; anything unknown falls back to the kit's own default glyph. */
  icon: string | null | undefined;
  /** Series slot — decides the tint. Ignored when `tint` is given. */
  index?: number;
  tint?: Tint;
  size?: number;
}

/** A category or group avatar: the glyph inside its tinted circle. */
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

export type BadgeTone = "neutral" | "accent" | "overspend";

/** ВЛАСНИК, шаблон, приховано — a word that qualifies the row it sits on. */
export function Badge({ label, tone = "neutral" }: { label: string; tone?: BadgeTone }) {
  const style =
    tone === "accent"
      ? "bg-accent-900 text-accent-300"
      : tone === "overspend"
        ? "text-overspend"
        : "text-neutral-500";
  return (
    <Text
      className={`rounded-sm px-[5px] py-[2px] text-[9.5px] font-medium uppercase ${style}`}
      style={{ letterSpacing: 0.6, backgroundColor: tone === "accent" ? undefined : "rgba(233,233,237,.08)" }}
    >
      {label}
    </Text>
  );
}

// ---- controls -----------------------------------------------------------------------------------

export interface ChipProps {
  label: string;
  /** The right-hand figure on a template chip — `Кава ₴50`. */
  amount?: Money;
  icon?: IconName;
  onPress?: () => void;
  /** The design's long-press affordance: tap logs, long-press opens the sheet prefilled. */
  onLongPress?: () => void;
  selected?: boolean;
  /** A chip that adds rather than picks — the dashed `Новий`. */
  variant?: "solid" | "outline";
  className?: string;
}

/** The template chip: a pill carrying a name and, usually, its amount. */
export function Chip({
  label,
  amount,
  icon,
  onPress,
  onLongPress,
  selected = false,
  variant = "solid",
  className = "",
}: ChipProps) {
  const shell =
    variant === "outline"
      ? "border border-dashed border-neutral-700 bg-transparent"
      : selected
        ? "bg-accent-800"
        : "bg-surface";
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected }}
      onPress={onPress}
      onLongPress={onLongPress}
      className={`flex-none flex-row items-center gap-n2 rounded-full px-n4 py-n3 ${shell} ${className}`}
      style={({ pressed }) => (pressed ? { opacity: 0.75 } : null)}
    >
      {icon ? <Icon name={icon} size={15} color={nocturne.accent[300]} /> : null}
      <Text className="text-[12.5px] font-medium text-fg">{label}</Text>
      {amount ? <MoneyText value={amount} size={12.5} tone="accent" /> : null}
    </Pressable>
  );
}

export interface ButtonProps {
  title: string;
  onPress: () => void;
  disabled?: boolean;
  /** `primary` fills with the accent; `ghost` is the quiet secondary the sheets use. */
  variant?: "primary" | "ghost";
  icon?: IconName;
  className?: string;
}

export function Button({ title, onPress, disabled = false, variant = "primary", icon, className = "" }: ButtonProps) {
  const shell = variant === "primary" ? "bg-accent" : "border border-neutral-700 bg-transparent";
  const text = variant === "primary" ? "text-bg" : "text-fg";
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      className={`flex-row items-center justify-center gap-n2 rounded-md px-n5 py-n4 ${shell} ${className}`}
      style={disabled ? { opacity: 0.45 } : undefined}
    >
      {icon ? <Icon name={icon} size={18} color={variant === "primary" ? nocturne.bg : nocturne.text} /> : null}
      <Text className={`text-[14.5px] font-semibold ${text}`}>{title}</Text>
    </Pressable>
  );
}

/** The floating add button. One per screen, bottom-right, above the gesture pill. */
export function Fab({ label, onPress, icon = "plus" }: { label: string; onPress: () => void; icon?: IconName }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      className="absolute bottom-[26px] right-[18px] h-[56px] w-[56px] items-center justify-center rounded-full bg-accent"
      style={({ pressed }) => ({
        opacity: pressed ? 0.85 : 1,
        shadowColor: "#000",
        shadowOpacity: 0.45,
        shadowRadius: 14,
        shadowOffset: { width: 0, height: 8 },
        elevation: 8,
      })}
    >
      <Icon name={icon} size={24} color={nocturne.bg} />
    </Pressable>
  );
}

export interface FieldProps extends Omit<TextInputProps, "className" | "style"> {
  label: string;
  /** Turns the underline and the message to `overspend`. */
  error?: string;
}

/** The underlined input from the onboarding screen — Nocturne has no boxed text field. */
export function Field({ label, error, ...input }: FieldProps) {
  return (
    <View>
      <Kicker className="mb-n2">{label}</Kicker>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={nocturne.neutral[600]}
        selectionColor={nocturne.accent.DEFAULT}
        className="pb-n2 text-[16px] text-fg"
        style={{ borderBottomWidth: 1, borderBottomColor: error ? nocturne.overspend : nocturne.neutral[700] }}
        {...input}
      />
      {error ? <Text className="mt-n2 text-[12px] text-overspend">{error}</Text> : null}
    </View>
  );
}

/** A settings toggle row — "Сповіщення про перевитрату". */
export function ToggleRow({
  label,
  subtitle,
  value,
  onValueChange,
  divider = true,
}: {
  label: string;
  subtitle?: string;
  value: boolean;
  onValueChange: (next: boolean) => void;
  divider?: boolean;
}) {
  return (
    <Row
      title={label}
      subtitle={subtitle}
      divider={divider}
      trailing={
        <Switch
          value={value}
          onValueChange={onValueChange}
          accessibilityLabel={label}
          trackColor={{ false: nocturne.neutral[800], true: nocturne.accent[700] }}
          thumbColor={value ? nocturne.accent[200] : nocturne.neutral[500]}
        />
      }
    />
  );
}

/** One figure with its label under it — the household screen's Витрати / Частка / Операцій. */
export function Stat({ label, value, tone = "default" }: { label: string; value: string; tone?: MoneyTone }) {
  return (
    <View className="flex-1">
      <Text className="text-[10.5px] uppercase text-neutral-600" style={{ letterSpacing: 0.8 }}>
        {label}
      </Text>
      <Text className={`mt-[3px] text-[15px] font-medium ${TONE_CLASS[tone]}`}>{value}</Text>
    </View>
  );
}

// ---- states -------------------------------------------------------------------------------------

export interface EmptyStateProps {
  title: string;
  body?: string;
  icon?: IconName;
  action?: { label: string; onPress: () => void };
  className?: string;
}

export function EmptyState({ title, body, icon = "tray", action, className = "" }: EmptyStateProps) {
  return (
    <View className={`items-center justify-center gap-n3 px-n6 py-[48px] ${className}`}>
      <View className="h-[44px] w-[44px] items-center justify-center rounded-md border border-accent">
        <Icon name={icon} size={22} color={nocturne.accent[400]} />
      </View>
      <Text className="text-center text-[17px] font-medium text-fg">{title}</Text>
      {body ? (
        <Text className="text-center text-[13px] leading-[20px] text-neutral-500">{body}</Text>
      ) : null}
      {action ? <Button title={action.label} onPress={action.onPress} className="mt-n3" /> : null}
    </View>
  );
}

// ---- sheet ----------------------------------------------------------------------------------------

export interface SheetProps {
  visible: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  /** A sheet whose body should scroll rather than grow past the screen. */
  scroll?: boolean;
}

/**
 * The bottom sheet: scrim, rounded top, grabber, then the caller's content. A plain
 * `Modal` — no gesture-driven sheet library, which would be a new stack row for one screen.
 */
export function Sheet({ visible, onClose, title, children, scroll = false }: SheetProps) {
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={title ?? "close"}
        onPress={onClose}
        className="flex-1"
        style={{ backgroundColor: nocturne.scrim }}
      />
      <View className="max-h-[86%] rounded-t-lg bg-surface pb-n5">
        <View className="items-center py-n3">
          <View className="h-[4px] w-[42px] rounded-full" style={{ backgroundColor: nocturne.neutral[700] }} />
        </View>
        {title ? (
          <Text className="px-n5 pb-n3 text-[17px] font-medium text-fg">{title}</Text>
        ) : null}
        {scroll ? <ScrollView className="px-n5">{children}</ScrollView> : <View className="px-n5">{children}</View>}
      </View>
    </Modal>
  );
}
