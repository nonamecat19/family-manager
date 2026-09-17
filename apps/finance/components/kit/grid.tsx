import { Icon } from "@fm/ui";
import { Pressable, Text, View } from "react-native";

import { organic, tintFor } from "./tokens.ts";
import { IconCircle } from "./ui.tsx";

export interface CategoryGridItem {
  id: string;
  label: string;
  icon?: string | null;
  color?: string;
}

export interface CategoryIconGridProps {
  items: readonly CategoryGridItem[];
  selectedId?: string;
  onSelect: (item: CategoryGridItem) => void;
  onLongPress?: (item: CategoryGridItem) => void;
  columns?: number;
  more?: { label: string; onPress: () => void };
  className?: string;
}

export function CategoryIconGrid({
  items,
  selectedId,
  onSelect,
  onLongPress,
  columns = 4,
  more,
  className = "",
}: CategoryIconGridProps) {
  const width: `${number}%` = `${100 / columns}%`;
  return (
    <View className={`flex-row flex-wrap ${className}`}>
      {items.map((item, index) => {
        const selected = item.id === selectedId;
        const tint = item.color ? { bg: item.color, fg: organic.text } : tintFor(index);
        return (
          <Pressable
            key={item.id}
            accessibilityRole="button"
            accessibilityLabel={item.label}
            accessibilityState={{ selected }}
            onPress={() => onSelect(item)}
            onLongPress={onLongPress ? () => onLongPress(item) : undefined}
            className="items-center py-2.1"
            style={{ width }}
          >
            <View
              style={{
                padding: 2,
                borderRadius: 999,
                borderWidth: selected ? 1.5 : 0,
                borderColor: organic.accent.DEFAULT,
              }}
            >
              <IconCircle icon={item.icon} tint={tint} size={42} />
            </View>
            <Text
              className={`mt-1.4 text-center text-10.5 ${selected ? "font-fig-bold text-fg" : "font-fig text-neutral-600"}`}
              numberOfLines={1}
            >
              {item.label}
            </Text>
          </Pressable>
        );
      })}
      {more ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={more.label}
          onPress={more.onPress}
          className="items-center py-2.1"
          style={{ width }}
        >
          <View className="h-[46px] w-[46px] items-center justify-center rounded-full border border-dashed border-neutral-400">
            <Icon name="dots-three" size={20} color={organic.neutral[600]} />
          </View>
          <Text className="mt-1.4 text-center font-fig text-10.5 text-neutral-600">{more.label}</Text>
        </Pressable>
      ) : null}
    </View>
  );
}
