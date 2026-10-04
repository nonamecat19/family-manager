import type { ReactNode } from "react";
import { Pressable, Switch, Text, View } from "react-native";

import { Icon, type IconName } from "./icons.tsx";
import { Kicker } from "./primitives.tsx";
import { useTheme } from "./theme.tsx";

export function SettingsSection({
  title,
  hint,
  children,
  className = "",
}: {
  title: string;
  hint?: string;
  children?: ReactNode;
  className?: string;
}) {
  const t = useTheme();
  return (
    <View className={className}>
      <Kicker className="mb-[11px]">{title}</Kicker>
      {hint ? (
        <Text
          className="mb-[12px] text-[14px] leading-[21px] text-neutral-600"
          style={{ fontFamily: t.fonts?.body }}
        >
          {hint}
        </Text>
      ) : null}
      {children}
    </View>
  );
}

export function SettingsGroup({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <View className={`rounded-2xl bg-neutral-100 px-[16px] ${className}`}>{children}</View>;
}

export function SettingsLinkRow({
  label,
  onPress,
  trailing,
  icon,
  iconTone = "neutral",
  meta,
  className = "",
}: {
  label: string;
  onPress: () => void;
  trailing?: ReactNode;
  icon?: IconName;
  iconTone?: "neutral" | "accent";
  meta?: string;
  className?: string;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      className={`flex-row items-center gap-[14px] rounded-2xl bg-neutral-100 px-[16px] py-[14px] ${className}`}
    >
      {icon ? (
        <Icon name={icon} size={18} color={iconTone === "accent" ? t.accent[600] : t.neutral[600]} />
      ) : null}
      <View className="flex-1">
        <Text className="text-[15.5px] text-fg" numberOfLines={1} style={{ fontFamily: t.fonts?.bold }}>
          {label}
        </Text>
        {meta ? (
          <Text
            className="mt-[2px] text-[12px] text-neutral-600"
            numberOfLines={1}
            style={{ fontFamily: t.fonts?.body }}
          >
            {meta}
          </Text>
        ) : null}
      </View>
      {trailing ?? <Icon name="forward" size={16} color={t.muted} />}
    </Pressable>
  );
}

export function SettingsChoiceRow({
  label,
  selected,
  onPress,
  divider = false,
}: {
  label: string;
  selected: boolean;
  onPress: () => void;
  divider?: boolean;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected }}
      onPress={onPress}
      className={`flex-row items-center justify-between py-[14px] ${divider ? "border-b border-divider" : ""}`}
    >
      <Text className="text-[15.5px] text-fg" style={{ fontFamily: t.fonts?.bold }}>
        {label}
      </Text>
      {selected ? (
        <View
          className="h-[22px] w-[22px] items-center justify-center rounded-full"
          style={{ backgroundColor: t.accent.DEFAULT }}
        >
          <CheckMark />
        </View>
      ) : null}
    </Pressable>
  );
}

function CheckMark() {
  const t = useTheme();
  return <Icon name="check" size={12} color={t.accentFg} width={3.4} />;
}

export function SettingsToggleRow({
  label,
  value,
  onValueChange,
  disabled = false,
  divider = false,
  indent = 0,
}: {
  label: string;
  value: boolean;
  onValueChange: (next: boolean) => void;
  disabled?: boolean;
  divider?: boolean;
  indent?: number;
}) {
  const t = useTheme();
  return (
    <View
      className={`flex-row items-center justify-between py-[14px] ${divider ? "border-b border-divider" : ""}`}
      style={indent > 0 ? { paddingLeft: indent } : undefined}
    >
      <Text
        className="flex-1 text-[15px] text-fg"
        numberOfLines={1}
        style={{ fontFamily: t.fonts?.bold }}
      >
        {label}
      </Text>
      <Switch
        value={value}
        onValueChange={onValueChange}
        disabled={disabled}
        trackColor={{ false: t.neutral[300], true: t.accent.DEFAULT }}
        thumbColor={t.neutral[100]}
      />
    </View>
  );
}
