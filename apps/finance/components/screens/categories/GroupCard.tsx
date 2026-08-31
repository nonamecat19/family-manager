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
  /** The stored icon key; the kit falls back for anything it cannot draw. */
  icon?: string | null;
  /** The group's slot on the accent ramp (finance.v1 CategoryGroup.color_step). */
  colorStep: number;
  /** "7 категорій · бюджет ₴6,000", already assembled by the screen from the dictionary. */
  meta: string;
  expanded: boolean;
  onToggle: () => void;
  categories: readonly CategoryGridItem[];
  /** Label of the trailing cell that adds a category to this group. */
  addLabel: string;
  onAddCategory: () => void;
  onSelectCategory: (item: CategoryGridItem) => void;
}

/**
 * One row of screen 04: the group header, and — when expanded — its categories as a
 * four-across icon grid. Collapsed and expanded are the same card, so the list does not
 * reflow into a different shape when a group opens.
 */
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
