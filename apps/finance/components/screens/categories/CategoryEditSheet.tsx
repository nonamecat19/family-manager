import {
  toDisplayError,
  useDeleteCategory,
  useMoveCategory,
  useUpdateCategory,
  type Category,
  type GroupNode,
} from "@fm/api";
import { useEffect, useState } from "react";
import { Alert, Pressable, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { IconCircle, ScrollSheet, tintFor } from "@/components/kit";
import { Button, Chip, Field, Kicker } from "@fm/ui";

import { GROUP_ICONS } from "./GroupEditSheet.tsx";

export interface CategoryEditSheetProps {
  category: Category | null;
  groups: readonly GroupNode[];
  onClose: () => void;
}

export function CategoryEditSheet({ category, groups, onClose }: CategoryEditSheetProps) {
  const { t } = useI18n();

  const [name, setName] = useState("");
  const [icon, setIcon] = useState("");
  const [groupId, setGroupId] = useState("");
  const [targetCategoryId, setTargetCategoryId] = useState("");
  const [error, setError] = useState<string | null>(null);

  const update = useUpdateCategory();
  const move = useMoveCategory();
  const remove = useDeleteCategory();

  useEffect(() => {
    if (!category) return;
    setName(category.name);
    setIcon(category.icon);
    setGroupId(category.groupId);
    setTargetCategoryId("");
    setError(null);
  }, [category]);

  const busy = update.isPending || move.isPending || remove.isPending;
  const colorStep = groups.find((node) => node.group?.id === groupId)?.group?.colorStep ?? 0;
  const others = groups.flatMap((node) => node.categories).filter((c) => c.id !== category?.id);

  const save = async () => {
    if (!category) return;
    setError(null);
    if (name.trim() === "") {
      setError(t("categories.nameRequired"));
      return;
    }
    try {
      await update.mutateAsync({ categoryId: category.id, name: name.trim(), icon });
      if (groupId !== category.groupId) {
        await move.mutateAsync({ categoryId: category.id, targetGroupId: groupId });
      }
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("categories.saveError")).message);
    }
  };

  const confirmDelete = () => {
    if (!category) return;
    Alert.alert(t("categories.deleteCategoryTitle"), t("categories.deleteCategoryBody", { name: category.name }), [
      { text: t("common.cancel"), style: "cancel" },
      {
        text: t("common.delete"),
        style: "destructive",
        onPress: () =>
          remove.mutate(
            { categoryId: category.id, reassignToCategoryId: targetCategoryId },
            {
              onSuccess: onClose,
              onError: (e) => setError(toDisplayError(e, t("categories.saveError")).message),
            },
          ),
      },
    ]);
  };

  return (
    <ScrollSheet visible={category !== null} onClose={onClose} title={t("categories.editCategory")}>
      <View className="gap-3.5 pt-1.4">
        <Field label={t("categories.name")} value={name} onChangeText={setName} autoCapitalize="sentences" />

        <View className="gap-[8px]">
          <Kicker>{t("categories.icon")}</Kicker>
          <View className="flex-row flex-wrap gap-[8px]">
            {GROUP_ICONS.map((option) => (
              <Pressable
                key={option}
                accessibilityRole="button"
                accessibilityLabel={option}
                accessibilityState={{ selected: icon === option }}
                onPress={() => setIcon(option)}
                style={{ opacity: icon === option ? 1 : 0.45 }}
              >
                <IconCircle icon={option} tint={tintFor(colorStep)} size={36} />
              </Pressable>
            ))}
          </View>
        </View>

        <View className="gap-[8px]">
          <Kicker>{t("categories.group")}</Kicker>
          <View className="flex-row flex-wrap gap-[8px]">
            {groups.map((node) => (
              <Chip
                key={node.group?.id}
                label={node.group?.name ?? ""}
                active={groupId === node.group?.id}
                onPress={() => setGroupId(node.group?.id ?? "")}
              />
            ))}
          </View>
        </View>

        {error ? <Text className="text-[12.5px] text-error">{error}</Text> : null}

        <Button title={t("common.save")} onPress={() => void save()} disabled={busy} />

        <View className="mt-[8px] gap-[8px] border-t border-divider pt-3">
          <Kicker>{t("categories.moveTransactionsTo")}</Kicker>
          <View className="flex-row flex-wrap gap-[8px]">
            {others.map((other) => (
              <Chip
                key={other.id}
                label={other.name}
                active={targetCategoryId === other.id}
                onPress={() => setTargetCategoryId(targetCategoryId === other.id ? "" : other.id)}
              />
            ))}
          </View>
          <Text className="text-[11px] text-neutral-600">{t("categories.moveTransactionsHint")}</Text>
          <Button title={t("categories.deleteCategory")} tone="danger" onPress={confirmDelete} disabled={busy} />
        </View>
      </View>
    </ScrollSheet>
  );
}
