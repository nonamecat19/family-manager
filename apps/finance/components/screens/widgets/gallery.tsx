import { minorUnits, type Money } from "@fm/api";
import type { ReactNode } from "react";
import { Text, View } from "react-native";

import { Kicker, formatCompact, formatMoney, nocturne } from "@/components/nocturne";

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

export function GalleryItem({
  title,
  size,
  children,
  className = "",
}: {
  title: string;
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
