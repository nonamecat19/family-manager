import {
  fromWire,
  toDisplayError,
  TransactionKind,
  useCategoryTree,
  useCreateCategory,
  useCreateCategoryGroup,
  useFamily,
  type GroupNode,
} from "@fm/api";
import { useState } from "react";
import { ScrollView, Text, View } from "react-native";
import { useRouter } from "expo-router";

import { useI18n } from "@/components/i18n";
import {
  Button,
  Drawer,
  EmptyState,
  formatMoney,
  Screen,
  ScreenHeader,
  SegmentedTabs,
  useDrawerItems,
  type CategoryGridItem,
  type SegmentedOption,
} from "@/components/nocturne";
import { DashedAction } from "@/components/screens/categories/DashedAction.tsx";
import { GroupCard } from "@/components/screens/categories/GroupCard.tsx";
import { NameSheet } from "@/components/screens/categories/NameSheet.tsx";

type Kind = "expense" | "income";

/** Which sheet is open, and what it will create when it submits. */
type Draft = { kind: "group" } | { kind: "category"; groupId: string; groupName: string };

const WIRE_KIND: Record<Kind, TransactionKind> = {
  expense: TransactionKind.EXPENSE,
  income: TransactionKind.INCOME,
};

/**
 * Screen 04 — Categories.
 *
 * The two-level taxonomy as the design draws it: a list of group cards, at most a few of
 * which are open, each opening into its own four-across icon grid. Nothing here paginates or
 * searches, because the grouping is the thing that keeps the list short.
 */
export default function CategoriesScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [kind, setKind] = useState<Kind>("expense");
  // null means "the screen has not been touched yet" — the first group is open, which is the
  // state the design shows. An explicit [] is a user who closed everything.
  const [openIds, setOpenIds] = useState<readonly string[] | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const drawerItems = useDrawerItems();
  const family = useFamily();

  const tree = useCategoryTree({ kind: WIRE_KIND[kind] });
  const createGroup = useCreateCategoryGroup();
  const createCategory = useCreateCategory();

  const groups: readonly GroupNode[] = tree.data ?? [];
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
    // The wire carries int64 minor units as bigint; `fromWire` is @fm/api's one crossing.
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
    <ScreenHeader
      title={t("categories.title")}
      gradient
      leading={{ icon: "list", label: t("nav.menu"), onPress: () => setMenuOpen(true) }}
      trailing={{
        icon: allOpen ? "caret-up" : "caret-down",
        label: allOpen ? t("common.none") : t("common.all"),
        onPress: () => setOpenIds(allOpen ? [] : ids),
      }}
    >
      <SegmentedTabs
        options={options}
        value={kind}
        onChange={(next) => {
          setKind(next);
          // Expansion is per taxonomy: the ids on the other side of the switch are different
          // rows, so carrying the set across would open arbitrary groups.
          setOpenIds(null);
        }}
        className="mt-n4"
      />
    </ScreenHeader>
  );

  return (
    <Screen>
      {header}

      {tree.isPending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[15px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
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
            icon="squares-four"
            title={t("categories.emptyTitle")}
            body={t("categories.emptyBody")}
            action={{ label: t("categories.newGroup"), onPress: () => setDraft({ kind: "group" }) }}
          />
        </View>
      ) : (
        <ScrollView
          className="flex-1"
          contentContainerClassName="gap-n3 px-n4 pb-n6 pt-n3"
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
                onAddCategory={() => setDraft({ kind: "category", groupId: group.id, groupName: group.name })}
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

      <Drawer
        visible={menuOpen}
        onClose={() => setMenuOpen(false)}
        account={{ name: family.data?.family?.name ?? "", email: "" }}
        household={{ name: family.data?.family?.name ?? "" }}
        items={drawerItems}
        activeId="categories"
        onSelect={(item) => {
          setMenuOpen(false);
          if (item.href && item.id !== "categories") router.push(item.href);
        }}
      />
    </Screen>
  );
}

/** The gate's error shape, repeated here because a failed tree must not blank the header. */
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
    <View className="flex-1 justify-center gap-n4 px-n6">
      <Text className="text-[13.5px] leading-[21px] text-neutral-500">{message}</Text>
      {reference ? <Text className="text-[12px] text-neutral-600">{referenceLabel(reference)}</Text> : null}
      <Button title={retryLabel} onPress={onRetry} />
    </View>
  );
}
