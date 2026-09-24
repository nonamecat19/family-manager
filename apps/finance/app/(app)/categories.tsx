import {
  CategoryGroupRole,
  fromWire,
  toDisplayError,
  TransactionKind,
  useCategoryTree,
  useCreateCategory,
  useCreateCategoryGroup,
  useFinanceSettings,
  type Category,
  type GroupNode,
} from "@fm/api";
import { useState } from "react";
import { ScrollView, Text, View } from "react-native";
import { useRouter } from "expo-router";

import { useI18n } from "@/components/i18n";
import { formatMoney, SegmentedTabs, type CategoryGridItem, type SegmentedOption } from "@/components/kit";
import { DashedAction } from "@/components/screens/categories/DashedAction.tsx";
import { CategoryEditSheet } from "@/components/screens/categories/CategoryEditSheet.tsx";
import { GroupCard } from "@/components/screens/categories/GroupCard.tsx";
import { GroupEditSheet } from "@/components/screens/categories/GroupEditSheet.tsx";
import { NameSheet } from "@/components/screens/categories/NameSheet.tsx";
import { EmptyState, IconButton, Screen, ScreenHeader, GateMessage } from "@fm/ui";

type Kind = "expense" | "income";

type Draft = { kind: "group" } | { kind: "category"; groupId: string; groupName: string };

const WIRE_KIND: Record<Kind, TransactionKind> = {
  expense: TransactionKind.EXPENSE,
  income: TransactionKind.INCOME,
};

export default function CategoriesScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [kind, setKind] = useState<Kind>("expense");
  const [openIds, setOpenIds] = useState<readonly string[] | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [editingGroupId, setEditingGroupId] = useState<string | null>(null);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const settings = useFinanceSettings();
  const currency = settings.data?.baseCurrencyCode || "UAH";

  const tree = useCategoryTree({ kind: WIRE_KIND[kind] });
  const createGroup = useCreateCategoryGroup();
  const createCategory = useCreateCategory();

  const groups: readonly GroupNode[] = tree.data ?? [];
  const editingGroup = groups.find((node) => node.group?.id === editingGroupId) ?? null;

  const managedScreen = (node: GroupNode) =>
    node.group?.role === CategoryGroupRole.INVESTMENTS
      ? "/(app)/investments"
      : node.group?.role === CategoryGroupRole.INSTALLMENTS
        ? "/(app)/installments"
        : null;
  const ids = groups.map((node) => node.group?.id ?? "");
  const open = openIds ?? ids.slice(0, 1);
  const allOpen = ids.length > 0 && open.length === ids.length;

  const toggle = (id: string) =>
    setOpenIds(open.includes(id) ? open.filter((each) => each !== id) : [...open, id]);

  const options: readonly SegmentedOption<Kind>[] = [
    { value: "expense", label: t("common.expenses") },
    { value: "income", label: t("common.income") },
  ];

  const groupMeta = (node: GroupNode): string => {
    const limit = node.budget?.budget?.limit ? fromWire(node.budget.budget.limit) : null;
    return t("categories.groupMeta", {
      categories: t("common.categoryCount", { count: node.categoryCount || node.categories.length }),
      budget:
        limit && limit.amountMinor > 0
          ? t("common.budgetOf", { amount: formatMoney(limit) })
          : t("common.noBudget"),
    });
  };

  const submitDraft = (name: string) => {
    if (!draft) return;
    setSaveError(null);
    const done = () => setDraft(null);
    const failed = (error: unknown) => setSaveError(toDisplayError(error, t("common.loadFailed")).message);
    if (draft.kind === "group") {
      createGroup.mutate({ name, kind: WIRE_KIND[kind] }, { onSuccess: done, onError: failed });
      return;
    }
    createCategory.mutate(
      { groupId: draft.groupId, name, kind: WIRE_KIND[kind] },
      { onSuccess: done, onError: failed },
    );
  };

  const header = (
    <View className="gap-[16px] px-[22px] pt-[8px]">
      <ScreenHeader
        title={t("categories.title")}
        onBack={() => router.back()}
        backLabel={t("common.back")}
        actions={
          <IconButton
            icon={allOpen ? "caret-up" : "caret-down"}
            label={allOpen ? t("common.none") : t("common.all")}
            onPress={() => setOpenIds(allOpen ? [] : ids)}
          />
        }
      />
      <SegmentedTabs
        options={options}
        value={kind}
        onChange={(next) => {
          setKind(next);
          setOpenIds(null);
        }}
      />
    </View>
  );

  return (
    <Screen>
      {header}

      {tree.isPending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[15px] text-neutral-600">{t("common.loadingEllipsis")}</Text>
        </View>
      ) : tree.isError ? (
        <ErrorPane
          message={toDisplayError(tree.error, t("common.loadFailed")).message}
          reference={toDisplayError(tree.error, t("common.loadFailed")).reference ?? null}
          retryLabel={t("common.tryAgain")}
          referenceLabel={(ref) => t("common.errorReference", { ref })}
          onRetry={() => void tree.refetch()}
        />
      ) : groups.length === 0 ? (
        <View className="flex-1 justify-center">
          <EmptyState
            title={t("categories.emptyTitle")}
            body={t("categories.emptyBody")}
            action={{ label: t("categories.newGroup"), onPress: () => setDraft({ kind: "group" }) }}
          />
        </View>
      ) : (
        <ScrollView
          className="flex-1"
          contentContainerClassName="gap-[8.4px] px-[11.2px] pb-[22.4px] pt-[8.4px]"
          showsVerticalScrollIndicator={false}
        >
          {groups.map((node, index) => {
            const group = node.group;
            if (!group) return null;
            const items: readonly CategoryGridItem[] = node.categories.map((category) => ({
              id: category.id,
              label: category.name,
              icon: category.icon,
            }));
            return (
              <GroupCard
                key={group.id}
                name={group.name}
                icon={group.icon}
                colorStep={group.colorStep || index}
                meta={groupMeta(node)}
                expanded={open.includes(group.id)}
                onToggle={() => toggle(group.id)}
                categories={items}
                addLabel={t("categories.addCategory")}
                onAddCategory={() => {
                  const managed = managedScreen(node);
                  if (managed) {
                    router.push(managed);
                    return;
                  }
                  setDraft({ kind: "category", groupId: group.id, groupName: group.name });
                }}
                editLabel={t("categories.editGroup")}
                onEdit={() => setEditingGroupId(group.id)}
                onLongPressCategory={(item) => {
                  const managed = managedScreen(node);
                  if (managed) {
                    router.push(managed);
                    return;
                  }
                  setEditingCategory(node.categories.find((category) => category.id === item.id) ?? null);
                }}
                onSelectCategory={(item) =>
                  router.push({
                    pathname: "/(app)/transactions",
                    params: { categoryId: item.id, kind },
                  })
                }
              />
            );
          })}

          <DashedAction label={t("categories.newGroup")} onPress={() => setDraft({ kind: "group" })} />
          <Text className="px-[8px] text-center text-[11px] text-neutral-600">{t("categories.editHint")}</Text>
        </ScrollView>
      )}

      <NameSheet
        visible={draft !== null}
        title={draft?.kind === "category" ? t("categories.addCategory") : t("categories.newGroup")}
        saveLabel={t("common.save")}
        cancelLabel={t("common.cancel")}
        submitting={createGroup.isPending || createCategory.isPending}
        error={saveError}
        onClose={() => {
          setDraft(null);
          setSaveError(null);
        }}
        onSubmit={submitDraft}
      />

      <GroupEditSheet
        node={editingGroup}
        otherGroups={groups.filter((node) => node.group?.id !== editingGroupId)}
        currency={currency}
        onClose={() => setEditingGroupId(null)}
      />

      <CategoryEditSheet
        category={editingCategory}
        groups={groups.filter((node) => !managedScreen(node))}
        onClose={() => setEditingCategory(null)}
      />
    </Screen>
  );
}

function ErrorPane({
  message,
  reference,
  retryLabel,
  referenceLabel,
  onRetry,
}: {
  message: string;
  reference: string | null;
  retryLabel: string;
  referenceLabel: (ref: string) => string;
  onRetry: () => void;
}) {
  return (
    <GateMessage
      body={message}
      reference={reference ? referenceLabel(reference) : undefined}
      actionTitle={retryLabel}
      onAction={onRetry}
    />
  );
}
