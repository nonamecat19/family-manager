import type { ReactNode } from "react";
import { Modal, Pressable, Text, View, type ViewProps } from "react-native";

import { useUiTranslate } from "./labels.tsx";
import { Display } from "./primitives.tsx";
import { useTheme } from "./theme.tsx";

function formatMacro(grams: number): string {
  return Number.isInteger(grams) ? String(grams) : grams.toFixed(1);
}

export function Sheet({
  visible,
  onClose,
  title,
  children,
}: {
  visible: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}) {
  const tLabel = useUiTranslate();
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <View className="flex-1 justify-end">
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={tLabel("common.close")}
          onPress={onClose}
          className="absolute inset-0 bg-scrim"
        />
        <View className="rounded-t-3xl bg-bg px-5.5 pb-8.5 pt-5">
          <View className="mb-4.5 h-1.25 w-[44px] self-center rounded-full bg-neutral-400" />
          <Display size={22} className="mb-lg">
            {title}
          </Display>
          {children}
        </View>
      </View>
    </Modal>
  );
}

export function Panel({ className = "", ...props }: ViewProps & { className?: string }) {
  return <View className={`rounded-2xl bg-neutral-100 ${className}`} {...props} />;
}

export function NutritionStrip({
  kcal,
  proteinG,
  fatG,
  carbsG,
  className = "",
}: {
  kcal: number;
  proteinG: number;
  fatG: number;
  carbsG: number;
  className?: string;
}) {
  const tLabel = useUiTranslate();
  const t = useTheme();
  if (kcal <= 0 && proteinG <= 0 && fatG <= 0 && carbsG <= 0) return null;
  const cells: { key: string; label: string; value: string }[] = [
    { key: "kcal", label: tLabel("nutrition.kcal"), value: String(Math.round(kcal)) },
    {
      key: "protein",
      label: tLabel("nutrition.protein"),
      value: tLabel("nutrition.grams", { value: formatMacro(proteinG) }),
    },
    {
      key: "fat",
      label: tLabel("nutrition.fat"),
      value: tLabel("nutrition.grams", { value: formatMacro(fatG) }),
    },
    {
      key: "carbs",
      label: tLabel("nutrition.carbs"),
      value: tLabel("nutrition.grams", { value: formatMacro(carbsG) }),
    },
  ];
  return (
    <Panel className={`flex-row px-1.5 py-3 ${className}`}>
      {cells.map((c) => (
        <View key={c.key} className="flex-1 items-center">
          <Text className="text-17 text-accent-800" style={{ fontFamily: t.fonts?.display }}>
            {c.value}
          </Text>
          <Text
            className="mt-0.5 text-11 uppercase tracking-[0.7px] text-neutral-600"
            style={{ fontFamily: t.fonts?.body }}
          >
            {c.label}
          </Text>
        </View>
      ))}
    </Panel>
  );
}
