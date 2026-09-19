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
  useDeleteTemplate,
  useFinanceSettings,
  useTemplates,
  useUpdateTemplate,
  type Money,
  type QuickTemplate,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Alert, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { Fab, formatMoney, IconCircle, ListSection, MoneyText, Row, ScrollSheet, SegmentedTabs } from "@/components/kit";
import { AmountRow } from "@/components/screens/add-transaction/AmountRow";
import {
  AccountSheet,
  CategorySheet,
} from "@/components/screens/add-transaction/pickers";
import { Button, EmptyState, Field, iconOr, Screen, ScreenHeader, ScrollBody } from "@fm/ui";

type Kind = "expense" | "income";

export default function TemplatesScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [editing, setEditing] = useState<QuickTemplate | null>(null);
  const [creating, setCreating] = useState(false);

  const templates = useTemplates();
  const settings = useFinanceSettings();

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
      <ScrollBody>
        <ScreenHeader
          title={t("templates.title")}
          onBack={() => router.back()}
          backLabel={t("common.back")}
        />

        {templates.isPending ? (
          <View className="items-center py-[28px]">
            <Text className="text-[13px] text-neutral-600">{t("common.loadingEllipsis")}</Text>
          </View>
        ) : templates.isError ? (
          <View className="gap-3 py-[18px]">
            <Text className="text-[13.5px] leading-[21px] text-neutral-600">
              {toDisplayError(templates.error, t("common.loadFailed")).message}
            </Text>
            <Button title={t("common.tryAgain")} onPress={() => void templates.refetch()} />
          </View>
        ) : list.length === 0 ? (
          <EmptyState
            title={t("templates.emptyTitle")}
            body={t("templates.emptyBody")}
            action={{ label: t("templates.new"), onPress: () => setCreating(true) }}
          />
        ) : (
          <>
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

            <Button
              title={t("templates.logFromAdd")}
              tone="quiet"
              onPress={() => router.push("/(app)/add")}
            />
          </>
        )}
      </ScrollBody>

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
    </Screen>
  );
}

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
      <ScrollSheet
        visible={visible}
        onClose={onClose}
        title={template ? t("templates.editTitle") : t("templates.new")}
        scroll
      >
        <View className="gap-2.8">
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
            tone="quiet"
            onPress={() => setPicker("account")}
          />
          <Button
            title={category ? category.name : t("add.category")}
            tone="quiet"
            onPress={() => setPicker("category")}
          />

          {error ? <Text className="text-[12.5px] text-error">{error}</Text> : null}

          <Button
            title={t("common.save")}
            onPress={() => void save()}
            disabled={create.isPending || update.isPending}
          />
          <Text className="text-center text-[11px] text-neutral-600">
            {t("templates.amountNote", { amount: amount ? formatMoney(amount) : "" })}
          </Text>
        </View>
      </ScrollSheet>

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
