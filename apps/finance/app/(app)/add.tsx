import {
  addDays,
  fromWire,
  MemberStatus,
  parseAmount,
  toDisplayError,
  toInput,
  toISODate,
  TransactionKind,
  TransactionType,
  useAccounts,
  useCategoryTree,
  useCreateTransaction,
  useFinanceMembers,
  useFinanceSettings,
  useLogTemplate,
  useTemplates,
  useTransaction,
  useUpdateTransaction,
  type Money,
  type QuickTemplate,
  type WireMoney,
} from "@fm/api";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  BottomIndicator,
  Button,
  CategoryIconGrid,
  formatMoney,
  iconOr,
  Kicker,
  Screen,
  ScreenHeader,
  SegmentedTabs,
  zeroLike,
  type CategoryGridItem,
} from "@/components/nocturne";

import { AmountRow } from "@/components/screens/add-transaction/AmountRow";
import { DateStrip } from "@/components/screens/add-transaction/DateStrip";
import { PickerField } from "@/components/screens/add-transaction/PickerField";
import { TemplateStrip } from "@/components/screens/add-transaction/TemplateStrip";
import {
  AccountSheet,
  CategorySheet,
  DateSheet,
  MemberSheet,
} from "@/components/screens/add-transaction/pickers";

type Kind = "expense" | "income";

const RECENT_DAYS = 14;

export default function AddTransactionScreen() {
  const { t } = useI18n();
  const router = useRouter();
  const params = useLocalSearchParams<{ templateId?: string; transactionId?: string }>();
  const editingId = typeof params.transactionId === "string" ? params.transactionId : "";
  const seedTemplateId = typeof params.templateId === "string" ? params.templateId : "";

  const [pickedKind, setPickedKind] = useState<Kind | null>(null);
  const [typedAmount, setTypedAmount] = useState<string | null>(null);
  const [pickedMemberId, setPickedMemberId] = useState("");
  const [pickedAccountId, setPickedAccountId] = useState("");
  const [pickedGroupId, setPickedGroupId] = useState<string | null>(null);
  const [pickedCategoryId, setPickedCategoryId] = useState<string | null>(null);
  const [pickedTemplateId, setPickedTemplateId] = useState<string | null>(null);
  const [pickedDay, setPickedDay] = useState("");
  const [sheet, setSheet] = useState<"member" | "account" | "category" | "date" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorRef, setErrorRef] = useState<string | null>(null);

  const settings = useFinanceSettings();
  const members = useFinanceMembers();
  const accounts = useAccounts();
  const templates = useTemplates();
  const existing = useTransaction(editingId);
  const createTransaction = useCreateTransaction();
  const updateTransaction = useUpdateTransaction();
  const logTemplate = useLogTemplate();

  const baseCurrency = settings.data?.baseCurrencyCode || "UAH";
  const seedTemplate = (templates.data ?? []).find((tpl) => tpl.id === seedTemplateId);
  const editingTx = editingId === "" ? null : (existing.data ?? null);

  const seed = useMemo(() => {
    const from = (
      type: TransactionType,
      amount: WireMoney | undefined,
      accountId: string,
      memberId: string,
      categoryId: string,
      occurredOn: string,
      templateId: string,
    ) => ({
      kind: (type === TransactionType.INCOME ? "income" : "expense") as Kind,
      amountText: amount ? toInput(fromWire(amount, baseCurrency)) : "",
      accountId,
      memberId,
      categoryId,
      occurredOn,
      templateId,
    });
    if (editingTx) {
      return from(
        editingTx.type,
        editingTx.amount,
        editingTx.accountId,
        editingTx.memberId,
        editingTx.categoryId,
        editingTx.occurredOn,
        "",
      );
    }
    if (seedTemplate) {
      return from(
        seedTemplate.type,
        seedTemplate.amount,
        seedTemplate.accountId,
        seedTemplate.memberId,
        seedTemplate.categoryId,
        "",
        seedTemplate.id,
      );
    }
    return null;
  }, [editingTx, seedTemplate, baseCurrency]);

  const kind = pickedKind ?? seed?.kind ?? "expense";
  const amountText = typedAmount ?? seed?.amountText ?? "";
  const templateId = pickedTemplateId ?? seed?.templateId ?? "";
  const occurredOn = pickedDay || seed?.occurredOn || toISODate(new Date());

  const tree = useCategoryTree({
    kind: kind === "expense" ? TransactionKind.EXPENSE : TransactionKind.INCOME,
  });

  const today = useMemo(() => toISODate(new Date()), []);
  const dayOptions = useMemo(() => [today, addDays(today, -1), addDays(today, -2)], [today]);
  const recentDays = useMemo(
    () => Array.from({ length: RECENT_DAYS }, (_, i) => addDays(today, -i)),
    [today],
  );

  const activeMembers = useMemo(
    () => (members.data ?? []).filter((m) => m.status === MemberStatus.ACTIVE),
    [members.data],
  );
  const shared = accounts.data?.shared ?? [];
  const privateOwn = accounts.data?.privateOwn ?? [];
  const selectableAccounts = useMemo(() => [...shared, ...privateOwn], [shared, privateOwn]);
  const groups = tree.data ?? [];

  const memberId = pickedMemberId || seed?.memberId || activeMembers[0]?.userId || "";
  const member = activeMembers.find((m) => m.userId === memberId);
  const memberIndex = Math.max(
    0,
    activeMembers.findIndex((m) => m.userId === memberId),
  );

  const accountId = pickedAccountId || seed?.accountId || selectableAccounts[0]?.id || "";
  const account = selectableAccounts.find((a) => a.id === accountId);

  const chosenCategoryId = pickedCategoryId ?? seed?.categoryId ?? "";
  const owningGroupId =
    groups.find((node) => node.categories.some((category) => category.id === chosenCategoryId))
      ?.group?.id ?? "";
  const categoryId = !tree.isSuccess || owningGroupId !== "" ? chosenCategoryId : "";
  const groupId = pickedGroupId || owningGroupId || groups[0]?.group?.id || "";
  const group = groups.find((node) => node.group?.id === groupId);

  const currencyCode = account?.currencyCode || settings.data?.baseCurrencyCode || "UAH";
  const amount: Money | null = parseAmount(amountText, currencyCode);

  const gridItems: readonly CategoryGridItem[] = useMemo(
    () =>
      (group?.categories ?? []).map((category) => ({
        id: category.id,
        label: category.name,
        icon: category.icon,
      })),
    [group],
  );

  const pending =
    settings.isPending ||
    members.isPending ||
    accounts.isPending ||
    tree.isPending ||
    templates.isPending ||
    (editingId !== "" && existing.isPending);
  const failed = [settings, members, accounts, tree, templates, existing].find((q) => q.isError);

  const prefill = (template: QuickTemplate) => {
    setPickedTemplateId(template.id);
    setPickedKind(template.type === TransactionType.INCOME ? "income" : "expense");
    const templateAmount = fromWire(template.amount, currencyCode);
    setTypedAmount(toInput(templateAmount));
    if (template.accountId !== "") setPickedAccountId(template.accountId);
    if (template.memberId !== "") setPickedMemberId(template.memberId);
    setPickedCategoryId(template.categoryId);
    const owner = groups.find((node) =>
      node.categories.some((category) => category.id === template.categoryId),
    );
    setPickedGroupId(owner?.group?.id ?? "");
    setError(null);
  };

  const log = async (template: QuickTemplate) => {
    setError(null);
    setErrorRef(null);
    try {
      await logTemplate.mutateAsync({ templateId: template.id, occurredOn });
      router.back();
    } catch (e) {
      const shown = toDisplayError(e, t("add.saveError"));
      setError(shown.message);
      setErrorRef(shown.reference ?? null);
    }
  };

  const submit = async () => {
    setError(null);
    setErrorRef(null);
    if (!amount || amount.amountMinor === 0) {
      setError(t("add.amountRequired"));
      return;
    }
    if (categoryId === "" || accountId === "") {
      setError(t("add.categoryRequired"));
      return;
    }
    const type = kind === "expense" ? TransactionType.EXPENSE : TransactionType.INCOME;
    try {
      if (editingId !== "") {
        await updateTransaction.mutateAsync({
          transactionId: editingId,
          type,
          accountId,
          categoryId,
          amount,
          occurredOn,
          memberId,
        });
      } else {
        await createTransaction.mutateAsync({
          type,
          accountId,
          categoryId,
          amount,
          occurredOn,
          memberId,
          templateId,
        });
      }
      router.back();
    } catch (e) {
      const shown = toDisplayError(e, t("add.saveError"));
      setError(shown.message);
      setErrorRef(shown.reference ?? null);
    }
  };

  const busy = createTransaction.isPending || updateTransaction.isPending || logTemplate.isPending;

  return (
    <Screen>
      <ScreenHeader
        title={editingId === "" ? t("add.title") : t("add.editTitle")}
        gradient
        leading={{ icon: "arrow-left", label: t("common.back"), onPress: () => router.back() }}
      >
        <SegmentedTabs<Kind>
          className="mt-n3"
          value={kind}
          onChange={(next) => {
            setPickedKind(next);
            setPickedCategoryId("");
            setPickedGroupId("");
            setPickedTemplateId("");
          }}
          options={[
            { value: "expense", label: t("common.expenses") },
            { value: "income", label: t("common.income") },
          ]}
        />
      </ScreenHeader>

      {failed ? (
        <View className="flex-1 justify-center gap-n3 px-n5">
          <Text className="text-[13.5px] leading-[21px] text-neutral-500">
            {toDisplayError(failed.error, t("common.loadFailed")).message}
          </Text>
          <Button
            title={t("common.tryAgain")}
            onPress={() => {
              void settings.refetch();
              void members.refetch();
              void accounts.refetch();
              void tree.refetch();
              void templates.refetch();
              if (editingId !== "") void existing.refetch();
            }}
          />
        </View>
      ) : pending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
        </View>
      ) : (
        <>
          <ScrollView
            className="flex-1 px-n5"
            contentContainerClassName="pb-n5 pt-n4 gap-n5"
            keyboardShouldPersistTaps="handled"
          >
            <AmountRow
              label={t("add.amount")}
              value={amountText}
              onChangeValue={(next) => {
                setTypedAmount(next);
                setPickedTemplateId("");
              }}
              currencyCode={currencyCode}
              invalid={error === t("add.amountRequired")}
            />

            <TemplateStrip
              templates={templates.data ?? []}
              selectedId={templateId}
              onLog={(template) => void log(template)}
              onPrefill={prefill}
              onNew={() => router.push("/(app)/templates")}
              title={t("add.templates")}
              newLabel={t("add.newTemplate")}
              hint={t("add.templateHint")}
              fallbackCurrency={currencyCode}
            />

            <View className="flex-row gap-n3">
              <PickerField
                label={t("add.who")}
                value={member?.displayName ?? t("common.none")}
                avatar={{ name: member?.displayName ?? "", index: memberIndex }}
                onPress={() => setSheet("member")}
              />
              <PickerField
                label={t("add.account")}
                value={account?.name ?? t("common.none")}
                icon={iconOr(account?.icon, "credit-card")}
                onPress={() => setSheet("account")}
              />
            </View>

            <View>
              <View className="mb-n2 flex-row items-baseline justify-between">
                <Kicker>{t("add.category")}</Kicker>
                <Pressable
                  accessibilityRole="button"
                  accessibilityLabel={group?.group?.name ?? t("add.more")}
                  onPress={() => setSheet("category")}
                  hitSlop={6}
                >
                  <Text className="text-[11px] text-accent-400">
                    {group?.group?.name ? `${group.group.name} ▸` : t("add.more")}
                  </Text>
                </Pressable>
              </View>
              <CategoryIconGrid
                items={gridItems}
                selectedId={categoryId}
                onSelect={(item) => {
                  setPickedCategoryId(item.id);
                  setError(null);
                }}
                more={{ label: t("add.more"), onPress: () => setSheet("category") }}
              />
            </View>

            <DateStrip
              value={occurredOn}
              options={dayOptions}
              today={today}
              onChange={setPickedDay}
              onOpenCalendar={() => setSheet("date")}
              calendarLabel={t("add.date")}
              t={t}
            />

            {error ? (
              <View className="gap-n1">
                <Text className="text-[12.5px] text-overspend">{error}</Text>
                {errorRef ? (
                  <Text className="text-[11px] text-neutral-600">
                    {t("common.errorReference", { ref: errorRef })}
                  </Text>
                ) : null}
              </View>
            ) : null}
          </ScrollView>

          <View className="px-n5 pb-n2 pt-n3">
            <Button
              title={t("add.submit", {
                amount: formatMoney(amount ?? zeroLike(undefined, currencyCode)),
              })}
              onPress={() => void submit()}
              disabled={busy}
            />
          </View>
        </>
      )}

      <MemberSheet
        visible={sheet === "member"}
        onClose={() => setSheet(null)}
        title={t("add.who")}
        members={activeMembers}
        selectedId={memberId}
        onSelect={(id) => {
          setPickedMemberId(id);
          setSheet(null);
        }}
      />
      <AccountSheet
        visible={sheet === "account"}
        onClose={() => setSheet(null)}
        title={t("add.account")}
        shared={shared}
        privateOwn={privateOwn}
        sharedLabel={t("accounts.sharedSection")}
        privateLabel={t("accounts.privateSection", { name: member?.displayName ?? "" })}
        selectedId={accountId}
        onSelect={(id) => {
          setPickedAccountId(id);
          setSheet(null);
        }}
      />
      <CategorySheet
        visible={sheet === "category"}
        onClose={() => setSheet(null)}
        title={t("add.category")}
        groups={groups}
        selectedCategoryId={categoryId}
        onSelect={(nextGroupId, nextCategoryId) => {
          setPickedGroupId(nextGroupId);
          setPickedCategoryId(nextCategoryId);
          setError(null);
          setSheet(null);
        }}
      />
      <DateSheet
        visible={sheet === "date"}
        onClose={() => setSheet(null)}
        title={t("add.date")}
        days={recentDays}
        selected={occurredOn}
        onSelect={(iso) => {
          setPickedDay(iso);
          setSheet(null);
        }}
        t={t}
      />

      <BottomIndicator />
    </Screen>
  );
}
