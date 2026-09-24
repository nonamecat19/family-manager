import { Pressable, Text, View } from "react-native";

import { Card, type CategoryGridItem, CategoryIconGrid, IconCircle, tintFor } from "@/components/kit";
import { Icon, IconButton, organic } from "@fm/ui";

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
  onLongPressCategory?: (item: CategoryGridItem) => void;
  editLabel?: string;
  onEdit?: () => void;
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
  onLongPressCategory,
  editLabel,
  onEdit,
}: GroupCardProps) {
  return (
    <Card padded={false} className="overflow-hidden border border-border">
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={name}
        accessibilityState={{ expanded }}
        onPress={onToggle}
        className="flex-row items-center gap-[11.2px] px-[11.2px] py-[11.2px]"
        style={({ pressed }) => (pressed ? { opacity: 0.82 } : null)}
      >
        <IconCircle icon={icon} tint={tintFor(colorStep)} size={34} />
        <View className="flex-1">
          <Text className="text-[14px] font-fig-med text-fg" numberOfLines={1}>
            {name}
          </Text>
          <Text className="mt-[2px] text-[10.5px] text-neutral-600" numberOfLines={1}>
            {meta}
          </Text>
        </View>
{onEdit ? (
          <IconButton icon="pencil-simple" label={editLabel ?? name} size={16} onPress={onEdit} />
        ) : null}
        <Icon
          name={expanded ? "caret-up" : "caret-down"}
          size={14}
          color={expanded ? organic.neutral[600] : organic.neutral[700]}
        />
      </Pressable>

      {expanded ? (
        <CategoryIconGrid
          items={categories}
          onSelect={onSelectCategory}
          onLongPress={onLongPressCategory}
          more={{ label: addLabel, onPress: onAddCategory }}
          className="px-[8.4px] pb-[11.2px]"
        />
      ) : null}
    </Card>
  );
}
