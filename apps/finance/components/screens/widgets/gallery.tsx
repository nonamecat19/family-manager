/**
 * The gallery's own furniture: the cell a preview sits in, and the one amount style the
 * kit does not cover.
 */
import { minorUnits, type Money } from "@fm/api";
import type { ReactNode } from "react";
import { Text, View } from "react-native";

import { Kicker, formatCompact, formatMoney, nocturne } from "@/components/nocturne";

/**
 * A home-screen widget is not a Card: the design gives it a tighter 16px radius, a hairline
 * ring instead of the card's flat edge, and a slightly translucent surface so it reads as
 * sitting ON a wallpaper rather than on the app's ground.
 */
export function WidgetFrame({ children }: { children: ReactNode }) {
  return (
    <View
      style={{
        borderRadius: 16,
        borderWidth: 1,
        borderColor: nocturne.neutral[800],
        backgroundColor: nocturne.surface,
        padding: nocturne.space.n4,
      }}
    >
      {children}
    </View>
  );
}

/** One gallery entry: the type's name, its cell footprint, and the preview itself. */
export function GalleryItem({
  title,
  size,
  children,
  className = "",
}: {
  title: string;
  /** Already formatted — "4×2". */
  size: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <View className={className}>
      <View className="mb-n3 flex-row items-baseline justify-between px-[2px]">
        <Kicker>{title}</Kicker>
        <Text className="text-[10px] text-neutral-700">{size}</Text>
      </View>
      <WidgetFrame>{children}</WidgetFrame>
    </View>
  );
}

export type WidgetTone = "default" | "muted" | "overspend";

const TONE: Record<WidgetTone, string> = {
  default: "text-fg",
  muted: "text-neutral-500",
  overspend: "text-overspend",
};

/**
 * An amount inside a widget. `MoneyText` is the app's single money renderer, but it has no
 * compact mode, and a widget two to four cells wide is the one place the design shortens —
 * `₴52.4K`, `−₴10.8K`. So this draws the kit's own `formatCompact`/`formatMoney` in the kit's
 * own tones rather than inventing a second grammar.
 *
 * `compactFrom` is the design's threshold, not a rounding preference: the accounts strip
 * prints `₴1,478` in full and shortens `−₴10,835`, so shortening starts at ten thousand
 * there and at once for the budget and member figures.
 */
export function WidgetAmount({
  value,
  size,
  weight = "medium",
  tone = "default",
  compactFrom = 0,
}: {
  value: Money;
  size: number;
  weight?: "regular" | "medium";
  tone?: WidgetTone;
  compactFrom?: number;
}) {
  const units = Math.abs(value.amountMinor) / 10 ** minorUnits(value.currencyCode);
  const text = units >= compactFrom ? formatCompact(value) : formatMoney(value);
  return (
    <Text
      className={`${weight === "medium" ? "font-medium" : "font-normal"} ${TONE[tone]}`}
      style={{ fontSize: size, lineHeight: Math.round(size * 1.25) }}
    >
      {text}
    </Text>
  );
}
