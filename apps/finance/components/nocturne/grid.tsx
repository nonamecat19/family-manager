import { Pressable, Text, View } from "react-native";

import { Icon } from "./icons.tsx";
import { nocturne, tintFor } from "./tokens.ts";
import { IconCircle } from "./ui.tsx";

export interface CategoryGridItem {
  id: string;
  label: string;
  /** The stored icon key; anything the kit cannot draw falls back to its default glyph. */
  icon?: string | null;
  /** Overrides the ramp slot the item's position would give it. */
  color?: string;
}

export interface CategoryIconGridProps {
  items: readonly CategoryGridItem[];
  selectedId?: string;
  onSelect: (item: CategoryGridItem) => void;
  /** The design draws four across inside the add sheet. */
  columns?: number;
  /** The trailing "Ще" cell that opens the full list. */
  more?: { label: string; onPress: () => void };
  className?: string;
}

/**
 * The category picker from the add sheet: a glyph in a tinted circle with its name under it.
 * The point of the two-level model is that this grid is never forty items long — it shows one
 * group's categories, with `more` for the rest.
 */
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
