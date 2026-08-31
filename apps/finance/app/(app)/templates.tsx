import {
  fromWire,
  parseAmount,
  toDisplayError,
  toInput,
  TransactionKind,
  TransactionType,
  useAccounts,
  useCategoryTree,
  useCreateTemplate,
  useFamily,
  useDeleteTemplate,
  useFinanceSettings,
  useTemplates,
  useUpdateTemplate,
  type Money,
  type QuickTemplate,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Alert, ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  Button,
  Drawer,
  EmptyState,
  Fab,
  Field,
  formatMoney,
  IconCircle,
  iconOr,
  ListSection,
  MoneyText,
  Row,
  Screen,
  ScreenHeader,
  SegmentedTabs,
  Sheet,
  useDrawerItems,
} from "@/components/nocturne";
import { AmountRow } from "@/components/screens/add-transaction/AmountRow";
import {
  AccountSheet,
  CategorySheet,
} from "@/components/screens/add-transaction/pickers";

type Kind = "expense" | "income";

/**
 * Шаблони — the management screen behind the drawer's "Шаблони" row.
 *
 * The design draws templates as chips on Home and at the top of the add sheet; both of those
 * log or prefill. This screen is the other half: the place a template is created, repriced or
 * thrown away. Tapping a row opens it for editing rather than logging it — a screen whose rows
 * silently spend money is not a screen anyone can browse.
 */
export default function TemplatesScreen() {
  const { t } = useI18n();
  const router = useRouter();
  const drawerItems = useDrawerItems();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<QuickTemplate | null>(null);
  const [creating, setCreating] = useState(false);

  const templates = useTemplates();
  const settings = useFinanceSettings();
  const family = useFamily();

  const currency = settings.data?.baseCurrencyCode || "UAH";
  const list = templates.data ?? [];

  const remove = useDeleteTemplate();
  const confirmRemove = (template: QuickTemplate) => {
    Alert.alert(t("templates.deleteTitle"), t("templates.deleteBody", { label: template.label }), [
      { text: t("common.cancel"), style: "cancel" },
      {
        text: t("common.delete"),
        style: "destructive",
        onPress: () => remove.mutate(template.id),
      },
    ]);
  };

  return (
    <Screen>
      <ScreenHeader
        gradient
        title={t("templates.title")}
        subtitle={t("templates.subtitle")}
        leading={{ icon: "list", label: t("nav.menu"), onPress: () => setDrawerOpen(true) }}
      />

      {templates.isPending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
        </View>
      ) : templates.isError ? (
        <View className="flex-1 justify-center gap-n3 px-n5">
          <Text className="text-[13.5px] leading-[21px] text-neutral-500">
            {toDisplayError(templates.error, t("common.loadFailed")).message}
          </Text>
          <Button title={t("common.tryAgain")} onPress={() => void templates.refetch()} />
        </View>
      ) : list.length === 0 ? (
        <View className="flex-1 justify-center">
          <EmptyState
            icon="lightning"
            title={t("templates.emptyTitle")}
            body={t("templates.emptyBody")}
            action={{ label: t("templates.new"), onPress: () => setCreating(true) }}
          />
        </View>
      ) : (
        <ScrollView className="flex-1 px-n4" contentContainerClassName="pb-[96px] pt-n4">
          <ListSection title={t("templates.yours")}>
            {list.map((template, index) => (
              <Row
                key={template.id}
                title={template.label}
                subtitle={t("templates.used", { count: template.usageCount })}
                leading={<IconCircle icon={iconOr(template.icon, "lightning")} />}
                trailing={
                  <MoneyText value={fromWire(template.amount, currency)} size={14} weight="medium" />
                }
                onPress={() => setEditing(template)}
                onLongPress={() => confirmRemove(template)}
                chevron
                divider={index < list.length - 1}
              />
            ))}
          </ListSection>

          <View className="mt-n4">
            <Button
              title={t("templates.logFromAdd")}
              variant="ghost"
              icon="plus"
              onPress={() => router.push("/(app)/add")}
            />
          </View>
        </ScrollView>
      )}

      <Fab label={t("templates.new")} onPress={() => setCreating(true)} />

      <TemplateSheet
        visible={creating || editing !== null}
        template={editing}
        currency={currency}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
      />

      <Drawer
        visible={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        // No RPC names the signed-in person, so the panel names the household instead.
        account={{ name: family.data?.family?.name ?? "", email: "" }}
        household={{ name: family.data?.family?.name ?? "" }}
        items={drawerItems}
        activeId="templates"
        onSelect={(item) => {
          setDrawerOpen(false);
          if (item.href && item.id !== "templates") router.push(item.href);
        }}
      />
    </Screen>
  );
}

/** Create and edit are the same form: an edit is a create whose fields arrive filled in. */
function TemplateSheet({
  visible,
  template,
  currency,
  onClose,
}: {
  visible: boolean;
  template: QuickTemplate | null;
  currency: string;
  onClose: () => void;
}) {
  const { t } = useI18n();

  const [label, setLabel] = useState("");
  const [amountText, setAmountText] = useState("");
  const [kind, setKind] = useState<Kind>("expense");
  const [accountId, setAccountId] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [picker, setPicker] = useState<"account" | "category" | null>(null);
  const [error, setError] = useState<string | null>(null);
  // The form is seeded once per opening, keyed by the row it was opened from: re-seeding on
  // every render would undo the typing in progress.
  const [seededFor, setSeededFor] = useState<string | null>(null);

  const accounts = useAccounts();
  const tree = useCategoryTree({
    kind: kind === "expense" ? TransactionKind.EXPENSE : TransactionKind.INCOME,
  });
  const create = useCreateTemplate();
  const update = useUpdateTemplate();

  const selectable = useMemo(
    () => [...(accounts.data?.shared ?? []), ...(accounts.data?.privateOwn ?? [])],
    [accounts.data],
  );
  const groups = tree.data ?? [];

  const openKey = visible ? (template?.id ?? "new") : null;
  if (openKey !== seededFor) {
    setSeededFor(openKey);
    setLabel(template?.label ?? "");
    setAmountText(template?.amount ? toInput(fromWire(template.amount, currency)) : "");
    setKind(template?.type === TransactionType.INCOME ? "income" : "expense");
    setAccountId(template?.accountId ?? "");
    setCategoryId(template?.categoryId ?? "");
    setError(null);
  }

  const account = selectable.find((a) => a.id === accountId) ?? selectable[0];
  const currencyCode = account?.currencyCode || currency;
  const amount: Money | null = parseAmount(amountText, currencyCode);
  const category = groups
    .flatMap((node) => node.categories)
    .find((c) => c.id === categoryId);

  const save = async () => {
    setError(null);
    if (label.trim() === "") {
      setError(t("templates.labelRequired"));
      return;
    }
    if (!amount || amount.amountMinor === 0) {
      setError(t("add.amountRequired"));
      return;
    }
    if (categoryId === "" || !account) {
      setError(t("add.categoryRequired"));
      return;
    }
    try {
      if (template) {
        await update.mutateAsync({
          templateId: template.id,
          label: label.trim(),
          amount,
          categoryId,
          accountId: account.id,
        });
      } else {
        await create.mutateAsync({
          label: label.trim(),
          amount,
          type: kind === "expense" ? TransactionType.EXPENSE : TransactionType.INCOME,
          categoryId,
          accountId: account.id,
        });
      }
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("templates.saveError")).message);
    }
  };

  return (
    <>
      <Sheet
        visible={visible}
        onClose={onClose}
        title={template ? t("templates.editTitle") : t("templates.new")}
        scroll
      >
        <View className="gap-n4">
          <Field label={t("templates.label")} value={label} onChangeText={setLabel} />

          <SegmentedTabs<Kind>
            value={kind}
            onChange={(next) => {
              setKind(next);
              setCategoryId("");
                      }}
            options={[
              { value: "expense", label: t("common.expenses") },
              { value: "income", label: t("common.income") },
            ]}
          />

          <AmountRow
            label={t("add.amount")}
            value={amountText}
            onChangeValue={setAmountText}
            currencyCode={currencyCode}
            invalid={error === t("add.amountRequired")}
          />

          <Button
            title={account ? account.name : t("add.account")}
            variant="ghost"
            icon="credit-card"
            onPress={() => setPicker("account")}
          />
          <Button
            title={category ? category.name : t("add.category")}
            variant="ghost"
            icon="squares-four"
            onPress={() => setPicker("category")}
          />

          {error ? <Text className="text-[12.5px] text-overspend">{error}</Text> : null}

          <Button
            title={t("common.save")}
            onPress={() => void save()}
            disabled={create.isPending || update.isPending}
          />
          <Text className="text-center text-[11px] text-neutral-500">
            {t("templates.amountNote", { amount: amount ? formatMoney(amount) : "" })}
          </Text>
        </View>
      </Sheet>

      <AccountSheet
        visible={picker === "account"}
        onClose={() => setPicker(null)}
        title={t("add.account")}
        shared={accounts.data?.shared ?? []}
        privateOwn={accounts.data?.privateOwn ?? []}
        sharedLabel={t("accounts.sharedSection")}
        privateLabel={t("accounts.privateSection", { name: "" })}
        selectedId={account?.id ?? ""}
        onSelect={(id) => {
          setAccountId(id);
          setPicker(null);
        }}
      />
      <CategorySheet
        visible={picker === "category"}
        onClose={() => setPicker(null)}
        title={t("add.category")}
        groups={groups}
        selectedCategoryId={categoryId}
        onSelect={(_groupId, nextCategoryId) => {
          setCategoryId(nextCategoryId);
          setPicker(null);
        }}
      />
    </>
  );
}
