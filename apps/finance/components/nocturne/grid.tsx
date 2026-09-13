import { Pressable, Text, View } from "react-native";

import { Icon } from "./icons.tsx";
import { nocturne, tintFor } from "./tokens.ts";
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
  columns?: number;
  more?: { label: string; onPress: () => void };
  className?: string;
}

export function CategoryIconGrid({
  items,
  selectedId,
  onSelect,
  columns = 4,
  more,
  className = "",
}: CategoryIconGridProps) {
  const width: `${number}%` = `${100 / columns}%`;
  return (
    <View className={`flex-row flex-wrap ${className}`}>
      {items.map((item, index) => {
        const selected = item.id === selectedId;
        const tint = item.color ? { bg: item.color, fg: nocturne.bg } : tintFor(index);
        return (
          <Pressable
            key={item.id}
            accessibilityRole="button"
            accessibilityLabel={item.label}
            accessibilityState={{ selected }}
            onPress={() => onSelect(item)}
            className="items-center py-n3"
            style={{ width }}
          >
            <View
              className="rounded-full"
              style={{
                padding: 2,
                borderWidth: selected ? 1.5 : 0,
                borderColor: nocturne.accent[200],
                borderRadius: 999,
              }}
            >
              <IconCircle icon={item.icon} tint={tint} size={42} />
            </View>
            <Text
              className={`mt-n2 text-center text-[10.5px] ${selected ? "text-fg" : "text-neutral-400"}`}
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
          className="items-center py-n3"
          style={{ width }}
        >
          <View
            className="h-[46px] w-[46px] items-center justify-center rounded-full border border-dashed"
            style={{ borderColor: nocturne.neutral[700] }}
          >
            <Icon name="dots-three" size={20} color={nocturne.neutral[500]} />
          </View>
          <Text className="mt-n2 text-center text-[10.5px] text-neutral-400">{more.label}</Text>
        </Pressable>
      ) : null}
    </View>
  );
}
