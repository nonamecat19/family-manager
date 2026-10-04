import type { ReactNode } from "react";
import { Pressable, ScrollView, Text, View, type ColorValue } from "react-native";

import { organicTheme, type Theme } from "@fm/theme";

import { Icon, type IconName } from "./icons.tsx";
import { Display } from "./primitives.tsx";
import { PrimaryButton, RoundButton } from "./controls.tsx";
import { useTheme } from "./theme.tsx";

export const APP_FONT_FACES = {
  body: "NunitoSans_400Regular",
  medium: "NunitoSans_500Medium",
  semibold: "NunitoSans_600SemiBold",
  bold: "NunitoSans_700Bold",
  heavy: "NunitoSans_800ExtraBold",
  display: "Alegreya_800ExtraBold",
} as const;

export const appTheme: Theme = { ...organicTheme, fonts: { ...APP_FONT_FACES } };

export function useTabScreenOptions() {
  const t = useTheme();
  return {
    headerShown: false,
    tabBarActiveTintColor: t.accent[700],
    tabBarInactiveTintColor: t.neutral[600],
    tabBarStyle: {
      backgroundColor: t.neutral[100],
      borderTopColor: t.neutral[300],
      borderTopWidth: 1,
      height: 78,
      paddingTop: 10,
      paddingBottom: 22,
      elevation: 0,
    },
    tabBarLabelStyle: {
      fontFamily: t.fonts?.heavy,
      fontSize: 11,
      letterSpacing: 0.2,
    },
  };
}

export function tabIcon(name: IconName) {
  return function TabIcon({ color }: { color: ColorValue }) {
    return <Icon name={name} size={25} color={color as string} width={2.4} />;
  };
}

export function ScreenHeader({
  title,
  onBack,
  backLabel = "Back",
  size = 28,
  kicker,
  actions,
}: {
  title: string;
  onBack?: () => void;
  backLabel?: string;
  size?: number;
  kicker?: string;
  actions?: ReactNode;
}) {
  const t = useTheme();
  return (
    <View className="flex-row items-center gap-[12px]">
      {onBack ? <RoundButton icon="back" label={backLabel} onPress={onBack} /> : null}
      <View className="flex-1">
        {kicker ? (
          <Text className="mb-[2px] text-[13px] text-neutral-600" style={{ fontFamily: t.fonts?.bold }}>
            {kicker}
          </Text>
        ) : null}
        <Display size={size}>{title}</Display>
      </View>
      {actions ? <View className="flex-row items-center gap-[8px]">{actions}</View> : null}
    </View>
  );
}

export function ScrollBody({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <ScrollView
      showsVerticalScrollIndicator={false}
      keyboardShouldPersistTaps="handled"
      contentContainerClassName={`gap-[20px] px-[22px] pb-[28px] pt-[8px] ${className}`}
    >
      {children}
    </ScrollView>
  );
}

export function GateMessage({
  title,
  body,
  reference,
  actionTitle,
  onAction,
}: {
  title?: string;
  body: string;
  reference?: string;
  actionTitle?: string;
  onAction?: () => void;
}) {
  const t = useTheme();
  return (
    <View className="flex-1 justify-center gap-[18px] px-[22px]">
      {title ? <Display size={30}>{title}</Display> : null}
      <Text className="text-[15.5px] leading-[23px] text-neutral-700" style={{ fontFamily: t.fonts?.body }}>
        {body}
      </Text>
      {reference ? (
        <Text className="text-[13px] leading-[19px] text-neutral-600" style={{ fontFamily: t.fonts?.body }}>
          {reference}
        </Text>
      ) : null}
      {actionTitle && onAction ? <PrimaryButton title={actionTitle} onPress={onAction} /> : null}
    </View>
  );
}

export function BootSplash({ title, label }: { title: string; label?: string }) {
  const t = useTheme();
  return (
    <View className="flex-1 items-center justify-center gap-[10px] bg-bg">
      <Display size={30}>{title}</Display>
      {label ? (
        <Text className="text-[13.5px] text-neutral-600" style={{ fontFamily: t.fonts?.bold }}>
          {label}
        </Text>
      ) : null}
    </View>
  );
}

export function StatTile({ value, label }: { value: string; label: string }) {
  const t = useTheme();
  return (
    <View className="flex-1 rounded-2xl bg-neutral-100 px-[14px] py-[15px]">
      <Text className="text-[24px] text-accent-700" style={{ fontFamily: t.fonts?.display }}>
        {value}
      </Text>
      <Text className="mt-[3px] text-[12px] text-neutral-600" style={{ fontFamily: t.fonts?.bold }}>
        {label}
      </Text>
    </View>
  );
}

export function PillButton({ title, onPress }: { title: string; onPress: () => void }) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      onPress={onPress}
      className="items-center rounded-full bg-neutral-200 py-[13px]"
    >
      <Text className="text-[14.5px] text-neutral-700" style={{ fontFamily: t.fonts?.bold }}>
        {title}
      </Text>
    </Pressable>
  );
}

export function DangerLink({ title, onPress }: { title: string; onPress: () => void }) {
  const t = useTheme();
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={title} onPress={onPress} className="items-center pt-[4px]">
      <Text className="text-[14px] text-error" style={{ fontFamily: t.fonts?.semibold }}>
        {title}
      </Text>
    </Pressable>
  );
}

export function ErrorText({ children }: { children: ReactNode }) {
  const t = useTheme();
  return (
    <Text className="text-[12px] text-error" style={{ fontFamily: t.fonts?.semibold }}>
      {children}
    </Text>
  );
}
