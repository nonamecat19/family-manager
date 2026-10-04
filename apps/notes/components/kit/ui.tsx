import type { ReactNode } from "react";
import { Text, View, type ViewProps } from "react-native";
import { Avatar } from "@fm/ui";

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
        <View
          key={`${name}-${i}`}
          className="items-center justify-center rounded-full border-[1.5px] border-bg"
          style={{ width: size, height: size, marginLeft: i === 0 ? undefined : -overlap }}
        >
          <Avatar name={name} size={size - 3} index={i} />
        </View>
      ))}
      {extra > 0 ? (
        <View
          accessibilityRole="image"
          accessibilityLabel={`+${extra}`}
          className="items-center justify-center rounded-full border-[1.5px] border-bg bg-neutral-200"
          style={{ width: size, height: size, marginLeft: -overlap }}
        >
          <Text className="font-fig-semi text-neutral-700" style={{ fontSize: size * 0.38 }}>
            +{extra}
          </Text>
        </View>
      ) : null}
    </View>
  );
}

export function Rail({ children, className = "", ...props }: ViewProps & { className?: string }) {
  return (
    <View className={`border-r border-neutral-300 bg-neutral-200 ${className}`} {...props}>
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
  const ground = tone === "bg" ? "bg-bg" : "bg-surface";
  return (
    <View className={`${ground} ${className}`} {...props}>
      {children}
    </View>
  );
}

export function Kbd({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <View
      className={`flex-none items-center justify-center rounded-md border border-neutral-300 bg-neutral-100 px-[6px] py-[2px] ${className}`}
    >
      <Text className="font-fig-med text-[11px] text-neutral-600">{children}</Text>
    </View>
  );
}
