import { ActivityIndicator, Pressable, Text, TextInput, View, type TextInputProps } from "react-native";
import type { ReactNode } from "react";

import type { Theme } from "@fm/theme";

import { useTheme } from "./theme.tsx";


export function Card({ children }: { children: ReactNode }) {
  const t = useTheme();
  return (
    <View
      style={{
        backgroundColor: t.surface,
        borderRadius: t.radius.md,
        borderWidth: 1,
        borderColor: t.divider,
        overflow: "hidden",
      }}
    >
      {children}
    </View>
  );
}

export function CardHeader({ title, action }: { title: string; action?: ReactNode }) {
  const t = useTheme();
  return (
    <View
      style={{
        flexDirection: "row",
        alignItems: "center",
        justifyContent: "space-between",
        paddingHorizontal: 16,
        paddingTop: 14,
        paddingBottom: 10,
        gap: 12,
      }}
    >
      <Text
        style={{ color: t.muted, fontSize: 12, fontFamily: t.fonts?.medium, ...labelCase(t, 0.8) }}
      >
        {title}
      </Text>
      {action}
    </View>
  );
}

export function Divider() {
  const t = useTheme();
  return <View style={{ height: 1, backgroundColor: t.divider }} />;
}

export function Row({ children }: { children: ReactNode }) {
  return (
    <View
      style={{
        flexDirection: "row",
        alignItems: "center",
        gap: 12,
        paddingHorizontal: 16,
        paddingVertical: 12,
      }}
    >
      {children}
    </View>
  );
}

export function Avatar({ name, index = 0 }: { name: string; index?: number }) {
  const t = useTheme();
  const tint = avatarTint(t, name, index);
  const first = name.trim()[0];
  return (
    <View
      style={{
        width: 34,
        height: 34,
        borderRadius: 17,
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: tint.bg,
      }}
    >
      <Text style={{ color: tint.fg, fontSize: 13, fontWeight: "600", fontFamily: t.fonts?.semibold }}>
        {first ? first.toUpperCase() : "?"}
      </Text>
    </View>
  );
}

function avatarTint(t: Theme, name: string, fallbackIndex: number): { bg: string; fg: string } {
  const key = name.trim().toLowerCase();
  const slots = [
    { bg: t.accent[800], fg: t.accent[200] },
    { bg: t.accent2[700], fg: t.accent2[100] },
    { bg: t.accent[700], fg: t.accent[100] },
    { bg: t.accent2[800], fg: t.accent2[200] },
    { bg: t.neutral[800], fg: t.neutral[200] },
  ];
  const slot = key === "" ? fallbackIndex : hash(key);
  return slots[slot % slots.length]!;
}

function hash(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return h;
}

export type BadgeTone = "neutral" | "accent" | "danger";

export function Badge({ label, tone = "neutral" }: { label: string; tone?: BadgeTone }) {
  const t = useTheme();
  const bg = tone === "accent" ? t.accent[800] : tone === "danger" ? t.danger : t.neutral[800];
  const fg = tone === "accent" ? t.accent[200] : tone === "danger" ? t.dangerFg : t.neutral[300];
  return (
    <View style={{ backgroundColor: bg, borderRadius: t.radius.sm, paddingHorizontal: 8, paddingVertical: 3 }}>
      <Text style={{ color: fg, fontSize: 11, fontWeight: "600", fontFamily: t.fonts?.semibold }}>{label}</Text>
    </View>
  );
}

export type ButtonTone = "primary" | "quiet" | "danger";

export interface ButtonProps {
  title: string;
  onPress: () => void;
  tone?: ButtonTone;
  disabled?: boolean;
  busy?: boolean;
}

export function Button({ title, onPress, tone = "primary", disabled = false, busy = false }: ButtonProps) {
  const t = useTheme();
  const inert = disabled || busy;
  const bg = tone === "primary" ? t.accent.DEFAULT : tone === "danger" ? t.danger : "transparent";
  const fg = tone === "primary" ? t.accentFg : tone === "danger" ? t.dangerFg : t.accent.DEFAULT;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      accessibilityState={{ disabled: inert, busy }}
      disabled={inert}
      onPress={onPress}
      style={{
        flexDirection: "row",
        alignItems: "center",
        justifyContent: "center",
        gap: 8,
        backgroundColor: bg,
        borderRadius: t.radius.md,
        borderWidth: tone === "quiet" ? 1 : 0,
        borderColor: t.divider,
        paddingHorizontal: 16,
        paddingVertical: 11,
        opacity: inert ? 0.5 : 1,
      }}
    >
      {busy ? <ActivityIndicator size="small" color={fg} /> : null}
      <Text style={{ color: fg, fontSize: 14, fontWeight: "600", fontFamily: t.fonts?.semibold }}>{title}</Text>
    </Pressable>
  );
}

export function InlineAction({
  label,
  onPress,
  tone = "accent",
  disabled = false,
}: {
  label: string;
  onPress: () => void;
  tone?: "accent" | "danger";
  disabled?: boolean;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      hitSlop={8}
      style={{ opacity: disabled ? 0.4 : 1 }}
    >
      <Text style={{ color: tone === "danger" ? t.danger : t.accent.DEFAULT, fontSize: 13, fontWeight: "600", fontFamily: t.fonts?.semibold }}>
        {label}
      </Text>
    </Pressable>
  );
}

export interface FieldProps extends Omit<TextInputProps, "style"> {
  label?: string;
  error?: string | null;
}

export function Field({ label, error, ...input }: FieldProps) {
  const t = useTheme();
  return (
    <View style={{ gap: 6 }}>
      {label ? (
        <Text style={{ color: t.muted, fontSize: 12, fontFamily: t.fonts?.medium, ...labelCase(t, 0.6) }}>
          {label}
        </Text>
      ) : null}
      <TextInput
        {...input}
        placeholderTextColor={t.neutral[600]}
        style={{
          color: t.text,
          fontSize: 15,
          fontFamily: t.fonts?.body,
          ...inputTreatment(t, error != null && error !== ""),
        }}
      />
      {error ? <Text style={{ color: t.danger, fontSize: 12, fontFamily: t.fonts?.body }}>{error}</Text> : null}
    </View>
  );
}

function labelCase(t: Theme, tracking: number) {
  return (t.labelCase ?? "uppercase") === "sentence"
    ? ({ textTransform: "none", letterSpacing: 0 } as const)
    : ({ textTransform: "uppercase", letterSpacing: tracking } as const);
}

function inputTreatment(t: Theme, invalid: boolean) {
  const edge = invalid ? t.danger : t.divider;
  switch (t.fieldStyle ?? "outline") {
    case "underline":
      return {
        borderBottomWidth: 1,
        borderColor: edge,
        paddingHorizontal: 0,
        paddingVertical: 10,
      } as const;
    case "filled":
      return {
        backgroundColor: t.neutral[100],
        borderRadius: t.radius.sm,
        borderWidth: 1,
        borderColor: invalid ? t.danger : t.neutral[300],
        paddingHorizontal: 16,
        paddingVertical: 12,
      } as const;
    default:
      return {
        backgroundColor: t.surface,
        borderRadius: t.radius.sm,
        borderWidth: 1,
        borderColor: invalid ? t.danger : t.neutral[800],
        paddingHorizontal: 14,
        paddingVertical: 11,
      } as const;
  }
}

export function Muted({ children }: { children: ReactNode }) {
  const t = useTheme();
  return <Text style={{ color: t.muted, fontSize: 13, fontFamily: t.fonts?.body }}>{children}</Text>;
}

export function Title({ children }: { children: ReactNode }) {
  const t = useTheme();
  return <Text style={{ color: t.text, fontSize: 15, fontWeight: "600", fontFamily: t.fonts?.semibold }}>{children}</Text>;
}


export function Screen({ children }: { children: ReactNode }) {
  const t = useTheme();
  return <View style={{ flex: 1, backgroundColor: t.bg }}>{children}</View>;
}

export function Heading({ children }: { children: ReactNode }) {
  const t = useTheme();
  return (
    <Text style={{ color: t.text, fontSize: 27, fontWeight: "500", lineHeight: 31, fontFamily: t.fonts?.display ?? t.fonts?.medium }}>{children}</Text>
  );
}

export function Body({ children }: { children: ReactNode }) {
  const t = useTheme();
  return <Text style={{ color: t.muted, fontSize: 13.5, lineHeight: 21, fontFamily: t.fonts?.body }}>{children}</Text>;
}

export function Caption({ children }: { children: ReactNode }) {
  const t = useTheme();
  return <Text style={{ color: t.neutral[600], fontSize: 12, fontFamily: t.fonts?.body }}>{children}</Text>;
}

export function TextLink({ label, onPress }: { label: string; onPress: () => void }) {
  const t = useTheme();
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={label} onPress={onPress}>
      <Text style={{ color: t.accent[400], fontSize: 13.5, fontWeight: "500", textAlign: "center", fontFamily: t.fonts?.medium }}>
        {label}
      </Text>
    </Pressable>
  );
}

export function Loading({ label }: { label?: string }) {
  const t = useTheme();
  return (
    <View style={{ flex: 1, alignItems: "center", justifyContent: "center", gap: 12, backgroundColor: t.bg }}>
      <ActivityIndicator size="small" color={t.accent.DEFAULT} />
      {label ? <Text style={{ color: t.neutral[500], fontSize: 13, fontFamily: t.fonts?.body }}>{label}</Text> : null}
    </View>
  );
}

export interface EmptyStateProps {
  title: string;
  body?: string;
  action?: { label: string; onPress: () => void };
}

export function EmptyState({ title, body, action }: EmptyStateProps) {
  return (
    <View style={{ alignItems: "center", gap: 10, paddingVertical: 32, paddingHorizontal: 24 }}>
      <Title>{title}</Title>
      {body ? <Body>{body}</Body> : null}
      {action ? <Button title={action.label} tone="quiet" onPress={action.onPress} /> : null}
    </View>
  );
}
