import { ActivityIndicator, Pressable, Text, TextInput, View, type TextInputProps } from "react-native";
import type { ReactNode } from "react";

import type { Theme } from "@fm/theme";

import { useTheme } from "./theme.tsx";

/**
 * The themed primitives every app draws from.
 *
 * They style from the theme object rather than NativeWind classes on purpose. A class like
 * `bg-surface` resolves through whichever Tailwind config the CONSUMING app compiled, so a
 * shared component would silently look different depending on who rendered it, and could not
 * be re-themed at runtime at all. Reading `theme.surface` makes the theme the single answer
 * and makes switching it actually repaint.
 *
 * This replaced a NativeWind-class set that had no consumers left: the apps had each forked
 * their own copy (Nocturne in finance and notes, Organic in recipes) because the shared one was
 * painted for a system none of them use any more. Reading roles from the theme is what lets
 * one set serve all three.
 */

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

/** The initial an avatar shows, with a tint derived from the person so it is stable. */
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

/**
 * A stable tint per person: the same name is the same colour on every screen, and two people
 * in one list are not the same colour. Hashed rather than indexed because a member list
 * reorders (someone leaves) and a colour that moves with the row reads as a different person.
 */
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

/** A borderless text action — "Rename", "Revoke" — sized to sit inside a row. */
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

/** How a label is set, per theme. See `Theme.labelCase`. */
function labelCase(t: Theme, tracking: number) {
  return (t.labelCase ?? "uppercase") === "sentence"
    ? ({ textTransform: "none", letterSpacing: 0 } as const)
    : ({ textTransform: "uppercase", letterSpacing: tracking } as const);
}

/**
 * The shape of an input, per theme. See `Theme.fieldStyle` — three apps draw three different
 * inputs, and unifying that would be a restyle, not a refactor.
 */
function inputTreatment(t: Theme, invalid: boolean) {
  const edge = invalid ? t.danger : t.divider;
  switch (t.fieldStyle ?? "outline") {
    case "underline":
      // A bare rule under the text: no box, no fill. Nocturne's finance screens.
      return {
        borderBottomWidth: 1,
        borderColor: edge,
        paddingHorizontal: 0,
        paddingVertical: 10,
      } as const;
    case "filled":
      // A filled pill on the app ground. Organic.
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

/* ------------------------------------------------------------------ screen & type --- */

/**
 * The app ground. Every screen sits on it and none paints its own background — that rule is
 * why a theme switch repaints the whole app instead of leaving islands behind.
 */
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

/** The quiet line under an error — a support reference, a hint. */
export function Caption({ children }: { children: ReactNode }) {
  const t = useTheme();
  return <Text style={{ color: t.neutral[600], fontSize: 12, fontFamily: t.fonts?.body }}>{children}</Text>;
}

/** A centred text action: "I don't have an account yet". */
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

/**
 * The pre-content state. Deliberately text-light: it renders before an app has finished
 * loading its own fonts, so it must read in the platform face.
 */
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
