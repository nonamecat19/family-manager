import {
  BudgetPeriod,
  fromWire,
  parseAmount,
  toDisplayError,
  toInput,
  toISODate,
  useCreateBudget,
  useDeleteBudget,
  useDeleteCategoryGroup,
  useUpdateBudget,
  useUpdateCategoryGroup,
  type GroupNode,
} from "@fm/api";
import { useEffect, useState } from "react";
import { Alert, Pressable, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { IconCircle, ScrollSheet, tintFor } from "@/components/kit";
import { AmountRow } from "@/components/screens/add-transaction/AmountRow";
import { Button, Chip, Field, Kicker, type IconName } from "@fm/ui";

export const GROUP_ICONS: readonly IconName[] = [
  "fork-knife",
  "bowl-food",
  "house-line",
  "car-profile",
  "game-controller",
  "device-mobile",
  "graduation-cap",
  "users-three",
  "credit-card",
  "piggy-bank",
  "trend-up",
  "currency-btc",
  "chart-donut",
  "lock-key",
];

const COLOR_STEPS = [0, 1, 2, 3, 4, 5, 6, 7] as const;

export interface GroupEditSheetProps {
  node: GroupNode | null;
  otherGroups: readonly GroupNode[];
  currency: string;
  onClose: () => void;
}

function firstOfMonth(): string {
  const now = new Date();
  return toISODate(new Date(now.getFullYear(), now.getMonth(), 1));
}

export function GroupEditSheet({ node, otherGroups, currency, onClose }: GroupEditSheetProps) {
  const { t } = useI18n();
  const group = node?.group;
  const budget = node?.budget?.budget;

  const [name, setName] = useState("");
  const [icon, setIcon] = useState("");
  const [colorStep, setColorStep] = useState(0);
  const [limitText, setLimitText] = useState("");
  const [targetGroupId, setTargetGroupId] = useState("");
  const [error, setError] = useState<string | null>(null);

  const updateGroup = useUpdateCategoryGroup();
  const deleteGroup = useDeleteCategoryGroup();
  const createBudget = useCreateBudget();
  const updateBudget = useUpdateBudget();
  const deleteBudget = useDeleteBudget();

  useEffect(() => {
    if (!group) return;
    setName(group.name);
    setIcon(group.icon);
    setColorStep(group.colorStep);
    setLimitText(budget?.limit ? toInput(fromWire(budget.limit, currency)) : "");
    setTargetGroupId("");
    setError(null);
  }, [group, budget, currency]);

  const busy =
    updateGroup.isPending ||
    deleteGroup.isPending ||
    createBudget.isPending ||
    updateBudget.isPending ||
    deleteBudget.isPending;
  const budgetCurrency = budget?.limit?.currencyCode || currency;
  const categoryCount = node ? node.categoryCount || node.categories.length : 0;

  const save = async () => {
    if (!group) return;
    setError(null);
    if (name.trim() === "") {
      setError(t("categories.nameRequired"));
      return;
    }
    const limit = limitText.trim() === "" ? null : parseAmount(limitText, budgetCurrency);
    if (limitText.trim() !== "" && (!limit || limit.amountMinor === 0)) {
      setError(t("add.amountRequired"));
      return;
    }
    try {
      await updateGroup.mutateAsync({ groupId: group.id, name: name.trim(), icon, colorStep });
      if (limit && budget) {
        await updateBudget.mutateAsync({ budgetId: budget.id, limit });
      } else if (limit) {
        await createBudget.mutateAsync({
          groupId: group.id,
          limit,
          period: BudgetPeriod.MONTH,
          startOn: firstOfMonth(),
        });
      } else if (budget) {
        await deleteBudget.mutateAsync(budget.id);
      }
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("categories.saveError")).message);
    }
  };

  const remove = () => {
    if (!group) return;
    if (categoryCount > 0 && targetGroupId === "") {
      setError(t("categories.chooseTargetGroup"));
      return;
    }
    Alert.alert(t("categories.deleteGroupTitle"), t("categories.deleteGroupBody", { name: group.name }), [
      { text: t("common.cancel"), style: "cancel" },
      {
        text: t("common.delete"),
        style: "destructive",
        onPress: () =>
          deleteGroup.mutate(
            { groupId: group.id, reassignToGroupId: targetGroupId },
            {
              onSuccess: onClose,
              onError: (e) => setError(toDisplayError(e, t("categories.saveError")).message),
            },
          ),
      },
    ]);
  };

  return (
    <ScrollSheet visible={node !== null} onClose={onClose} title={t("categories.editGroup")}>
      <View className="gap-3.5 pt-1.4">
        <Field label={t("categories.name")} value={name} onChangeText={setName} autoCapitalize="sentences" />

        <View className="gap-sm">
          <Kicker>{t("categories.icon")}</Kicker>
          <View className="flex-row flex-wrap gap-sm">
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

        <View className="gap-sm">
          <Kicker>{t("categories.color")}</Kicker>
          <View className="flex-row gap-sm">
            {COLOR_STEPS.map((step) => (
              <Pressable
                key={step}
                accessibilityRole="button"
                accessibilityLabel={t("categories.colorN", { n: step + 1 })}
                accessibilityState={{ selected: colorStep === step }}
                onPress={() => setColorStep(step)}
                className="h-[30px] w-[30px] rounded-full"
                style={{
                  backgroundColor: tintFor(step).bg,
                  borderWidth: colorStep === step ? 2 : 0,
                  borderColor: tintFor(step).fg,
                }}
              />
            ))}
          </View>
        </View>

        <View className="gap-1.5">
          <Kicker>{t("categories.monthlyBudget")}</Kicker>
          <AmountRow
            label={t("categories.monthlyBudget")}
            value={limitText}
            onChangeValue={setLimitText}
            currencyCode={budgetCurrency}
          />
          <Text className="text-center text-11 text-neutral-600">{t("categories.budgetHint")}</Text>
        </View>

        {error ? <Text className="text-12.5 text-error">{error}</Text> : null}

        <Button title={t("common.save")} onPress={() => void save()} disabled={busy} />

        <View className="mt-sm gap-sm border-t border-divider pt-3">
          {categoryCount > 0 ? (
            <>
              <Kicker>{t("categories.moveCategoriesTo")}</Kicker>
              <View className="flex-row flex-wrap gap-sm">
                {otherGroups.map((other) => (
                  <Chip
                    key={other.group?.id}
                    label={other.group?.name ?? ""}
                    active={targetGroupId === other.group?.id}
                    onPress={() => setTargetGroupId(other.group?.id ?? "")}
                  />
                ))}
              </View>
            </>
          ) : null}
          <Button title={t("categories.deleteGroup")} tone="danger" onPress={remove} disabled={busy} />
        </View>
      </View>
    </ScrollSheet>
  );
}
