import { Pressable, Text, View } from "react-native";

import {
  Card,
  CategoryIconGrid,
  Icon,
  IconCircle,
  nocturne,
  tintFor,
  type CategoryGridItem,
} from "@/components/nocturne";

export interface GroupCardProps {
  name: string;
  icon?: string | null;
  colorStep: number;
  meta: string;
  expanded: boolean;
  onToggle: () => void;
  categories: readonly CategoryGridItem[];
  addLabel: string;
  onAddCategory: () => void;
  onSelectCategory: (item: CategoryGridItem) => void;
}

export function GroupCard({
  name,
  icon,
  colorStep,
  meta,
  expanded,
  onToggle,
  categories,
  addLabel,
  onAddCategory,
  onSelectCategory,
}: GroupCardProps) {
  return (
    <Card padded={false} className="overflow-hidden border border-border">
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={name}
        accessibilityState={{ expanded }}
        onPress={onToggle}
        className="flex-row items-center gap-n4 px-n4 py-n4"
        style={({ pressed }) => (pressed ? { opacity: 0.82 } : null)}
      >
        <IconCircle icon={icon} tint={tintFor(colorStep)} size={34} />
        <View className="flex-1">
          <Text className="text-[14px] font-medium text-fg" numberOfLines={1}>
            {name}
          </Text>
          <Text className="mt-[2px] text-[10.5px] text-neutral-600" numberOfLines={1}>
            {meta}
          </Text>
        </View>
        <Icon
          name={expanded ? "caret-up" : "caret-down"}
          size={14}
          color={expanded ? nocturne.neutral[500] : nocturne.neutral[700]}
        />
      </Pressable>

      {expanded ? (
        <CategoryIconGrid
          items={categories}
          onSelect={onSelectCategory}
          more={{ label: addLabel, onPress: onAddCategory }}
          className="px-n3 pb-n4"
        />
      ) : null}
    </Card>
  );
}
