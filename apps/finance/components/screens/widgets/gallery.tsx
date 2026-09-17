import { minorUnits, type Money } from "@fm/api";
import type { ReactNode } from "react";
import { Text, View } from "react-native";

import { formatCompact, formatMoney } from "@/components/kit";
import { Kicker, organic } from "@fm/ui";

export function WidgetFrame({ children }: { children: ReactNode }) {
  return (
    <View
      style={{
        borderRadius: 16,
        borderWidth: 1,
        borderColor: organic.neutral[800],
        backgroundColor: organic.surface,
        padding: 11.2,
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
      <View className="mb-2.1 flex-row items-baseline justify-between px-0.5">
        <Kicker>{title}</Kicker>
        <Text className="text-10 text-neutral-700">{size}</Text>
      </View>
      <WidgetFrame>{children}</WidgetFrame>
    </View>
  );
}

export type WidgetTone = "default" | "muted" | "overspend";

const TONE: Record<WidgetTone, string> = {
  default: "text-fg",
  muted: "text-neutral-600",
  overspend: "text-error",
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
      className={`${weight === "medium" ? "font-fig-med" : "font-fig"} ${TONE[tone]}`}
      style={{ fontSize: size, lineHeight: Math.round(size * 1.25) }}
    >
      {text}
    </Text>
  );
}
