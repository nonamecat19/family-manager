import type { ReactNode } from "react";
import { Pressable, Text, View, type ViewProps } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import Svg, { Defs, LinearGradient, Rect, Stop } from "react-native-svg";

import { Icon, type IconName } from "./icons.tsx";
import { initialOf, nocturne, tintFor, type Tint } from "./tokens.ts";



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

export function Rail({ children, className = "", ...props }: ViewProps & { className?: string }) {
  return (
    <View className={`bg-rail ${className}`} {...props}>
      {children}
    </View>
  );
}

export function Pane({
  children,
  tone = "bg",
  className = "",
  ...props
}: ViewProps & { tone?: "bg" | "list" | "surface"; className?: string }) {
  const ground = tone === "list" ? "bg-list" : tone === "surface" ? "bg-surface" : "bg-bg";
  return (
    <View className={`${ground} ${className}`} {...props}>
      {children}
    </View>
  );
}

export function Divider({
  orientation = "horizontal",
  className = "",
  color = nocturne.divider,
}: {
  orientation?: "horizontal" | "vertical";
  className?: string;
  color?: string;
}) {
  const vertical = orientation === "vertical";
  const id = vertical ? "nocturne-rule-v" : "nocturne-rule-h";
  return (
    <View
      pointerEvents="none"
      className={`${vertical ? "w-[1px] self-stretch" : "h-[1px] w-full"} ${className}`}
    >
      <Svg width="100%" height="100%">
        <Defs>
          <LinearGradient id={id} x1="0" y1="0" x2={vertical ? "0" : "1"} y2={vertical ? "1" : "0"}>
            <Stop offset="0" stopColor={color} stopOpacity={0} />
            <Stop offset="0.5" stopColor={color} stopOpacity={1} />
            <Stop offset="1" stopColor={color} stopOpacity={0} />
          </LinearGradient>
        </Defs>
        <Rect x="0" y="0" width="100%" height="100%" fill={`url(#${id})`} />
      </Svg>
    </View>
  );
}


export interface CardProps {
  children: ReactNode;
  onPress?: () => void;
  onLongPress?: () => void;
  accessibilityLabel?: string;
  padded?: boolean;
  selected?: boolean;
  className?: string;
}

export function Card({
  children,
  onPress,
  onLongPress,
  accessibilityLabel,
  padded = true,
  selected = false,
  className = "",
}: CardProps) {
  const base = `rounded-lg bg-surface ${padded ? "p-4" : ""} ${
    selected ? "border border-accent-600" : ""
  } ${className}`;
  if (!onPress && !onLongPress) return <View className={base}>{children}</View>;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      accessibilityState={{ selected }}
      onPress={onPress}
      onLongPress={onLongPress}
      className={base}
      style={({ pressed }) => (pressed ? { opacity: 0.82 } : null)}
    >
      {children}
    </Pressable>
  );
}


export interface AvatarProps {
  name: string;
  size?: number;
  tint?: Tint;
  index?: number;
  ringed?: boolean;
}

export function Avatar({ name, size = 26, tint, index = 0, ringed = false }: AvatarProps) {
  const paint = tint ?? tintFor(name, index);
  return (
    <View
      accessibilityRole="image"
      accessibilityLabel={name}
      className="flex-none items-center justify-center"
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        backgroundColor: paint.bg,
        borderWidth: ringed ? 1.5 : 0,
        borderColor: nocturne.bg,
      }}
    >
      <Text style={{ fontSize: size * 0.42, color: paint.fg }} className="font-semi">
        {initialOf(name)}
      </Text>
    </View>
  );
}

export function AvatarStack({
  names,
  size = 24,
  max = 3,
  className = "",
}: {
  names: readonly string[];
  size?: number;
  max?: number;
  className?: string;
}) {
  if (names.length === 0) return null;
  const shown = names.slice(0, max);
  const extra = names.length - shown.length;
  const overlap = Math.round(size * 0.32);
  return (
    <View className={`flex-none flex-row items-center ${className}`}>
      {shown.map((name, i) => (
        <View key={`${name}-${i}`} style={i === 0 ? undefined : { marginLeft: -overlap }}>
          <Avatar name={name} size={size} index={i} ringed />
        </View>
      ))}
      {extra > 0 ? (
        <View
          accessibilityRole="image"
          accessibilityLabel={`+${extra}`}
          className="flex-none items-center justify-center bg-neutral-800"
          style={{
            width: size,
            height: size,
            borderRadius: size / 2,
            marginLeft: -overlap,
            borderWidth: 1.5,
            borderColor: nocturne.bg,
          }}
        >
          <Text style={{ fontSize: size * 0.38 }} className="font-semi text-neutral-300">
            +{extra}
          </Text>
        </View>
      ) : null}
    </View>
  );
}


export interface ChipProps {
  label: string;
  onPress?: () => void;
  active?: boolean;
  icon?: IconName;
  tone?: "accent" | "accent2" | "neutral";
  className?: string;
}

export function Chip({ label, onPress, active = false, icon, tone = "accent", className = "" }: ChipProps) {
  const fill =
    tone === "accent2" ? "border-accent2-600 bg-accent2-900" : tone === "neutral" ? "border-neutral-700 bg-neutral-900" : "border-accent-600 bg-accent-900";
  const text = tone === "accent2" ? "text-accent2-300" : tone === "neutral" ? "text-neutral-200" : "text-accent-300";
  const body = (
    <>
      {icon ? <Icon name={icon} size={13} color={active ? nocturne.accent[300] : nocturne.neutral[500]} /> : null}
      <Text className={`font-med text-[12.5px] ${active ? text : "text-neutral-400"}`}>{label}</Text>
    </>
  );
  const shape = `flex-none flex-row items-center gap-[5px] rounded-md border px-[9px] py-[5px] ${
    active ? fill : "border-neutral-800 bg-transparent"
  } ${className}`;

  if (!onPress) return <View className={shape}>{body}</View>;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected: active }}
      onPress={onPress}
      className={shape}
    >
      {body}
    </Pressable>
  );
}

export function Kbd({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <View
      className={`flex-none items-center justify-center rounded-sm border border-neutral-800 bg-neutral-900 px-[5px] py-[2px] ${className}`}
    >
      <Text className="font-med text-[11px] text-neutral-400">{children}</Text>
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
  color = nocturne.text,
  weight = "regular",
  badge = false,
  disabled = false,
}: IconButtonProps) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      hitSlop={8}
      className={`h-9 w-9 flex-none items-center justify-center rounded-md ${disabled ? "opacity-40" : ""}`}
    >
      <Icon name={icon} size={size} color={color} weight={weight} />
      {badge ? (
        <View className="absolute right-[7px] top-[7px] h-[7px] w-[7px] rounded-full bg-accent" />
      ) : null}
    </Pressable>
  );
}

export function PrimaryButton({
  title,
  onPress,
  icon,
  disabled = false,
  className = "",
}: {
  title: string;
  onPress: () => void;
  icon?: IconName;
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
      className={`flex-row items-center justify-center gap-[7px] rounded-md border border-accent bg-accent-900 px-4 py-[10px] ${
        disabled ? "opacity-45" : ""
      } ${className}`}
      style={({ pressed }) => (pressed ? { opacity: 0.8 } : null)}
    >
      {icon ? <Icon name={icon} size={16} color={nocturne.accent[300]} /> : null}
      <Text className="font-semi text-[14px] text-accent-300">{title}</Text>
    </Pressable>
  );
}


export interface EmptyStateProps {
  title: string;
  body?: string;
  icon?: IconName;
  action?: { label: string; onPress: () => void };
  className?: string;
}

export function EmptyState({ title, body, icon = "file-text", action, className = "" }: EmptyStateProps) {
  return (
    <View className={`items-center justify-center gap-3 px-6 py-[48px] ${className}`}>
      <View className="h-[44px] w-[44px] items-center justify-center rounded-md border border-accent-700">
        <Icon name={icon} size={22} color={nocturne.accent[400]} />
      </View>
      <Text className="text-center font-med text-[16px] text-fg">{title}</Text>
      {body ? (
        <Text className="text-center font-sans text-[13px] leading-[20px] text-neutral-500">{body}</Text>
      ) : null}
      {action ? <PrimaryButton title={action.label} onPress={action.onPress} className="mt-1" /> : null}
    </View>
  );
}

export { nocturne };
