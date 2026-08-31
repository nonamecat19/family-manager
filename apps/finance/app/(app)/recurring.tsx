import {
  fromWire,
  isISODate,
  parseAmount,
  RecurrenceUnit,
  toDisplayError,
  toISODate,
  TransactionKind,
  TransactionType,
  useAccounts,
  useCategoryTree,
  useCreateRecurringPayment,
  useDeleteRecurringPayment,
  useFamily,
  useFinanceSettings,
  usePostRecurringOccurrence,
  useRecurringPayments,
  useSkipRecurringOccurrence,
  useUpdateRecurringPayment,
  type Money,
  type RecurringPayment,
  type RecurringPaymentStatus,
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
  IconCircle,
  ListSection,
  MoneyText,
  Row,
  Screen,
  ScreenHeader,
  SegmentedTabs,
  shortDate,
  Sheet,
  ToggleRow,
  useDrawerItems,
  type Translate,
} from "@/components/nocturne";
import { AmountRow } from "@/components/screens/add-transaction/AmountRow";
import { AccountSheet, CategorySheet } from "@/components/screens/add-transaction/pickers";

/**
 * Регулярні платежі — the drawer's third row.
 *
 * A schedule is not a transaction: nothing is in the ledger until an occurrence is posted.
 * So the screen's job is the decision on a due one — "заплатили" or "пропустити" — and the
 * two are drawn side by side on the row rather than behind a menu. Posting twice for the same
 * due date is a server-side no-op, which is what makes a double tap harmless.
 */
export default function RecurringScreen() {
  const { t } = useI18n();
  const router = useRouter();
  const drawerItems = useDrawerItems();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [showInactive, setShowInactive] = useState(false);

  const payments = useRecurringPayments(showInactive);
  const settings = useFinanceSettings();
  const family = useFamily();

  const post = usePostRecurringOccurrence();
  const skip = useSkipRecurringOccurrence();
  const update = useUpdateRecurringPayment();
  const remove = useDeleteRecurringPayment();

  const currency = settings.data?.baseCurrencyCode || "UAH";
  // ListRecurringPayments answers with a status per schedule — the payment plus whether this
  // occurrence is already overdue, which the server decides in the household's timezone.
  const list = (payments.data ?? []).filter((status) => status.payment != null);
  const due = list.filter((status) => status.overdue);
  const upcoming = list.filter((status) => !status.overdue);

  const confirmRemove = (payment: RecurringPayment) => {
    Alert.alert(t("recurring.deleteTitle"), t("recurring.deleteBody", { name: payment.name }), [
      { text: t("common.cancel"), style: "cancel" },
      { text: t("common.delete"), style: "destructive", onPress: () => remove.mutate(payment.id) },
    ]);
  };

  const row = (status: RecurringPaymentStatus, isDue: boolean, last: boolean) => {
    const payment = status.payment as RecurringPayment;
    return (
    <Row
      key={payment.id}
      title={payment.name}
      subtitle={cadenceLabel(t, payment, currency)}
      leading={<IconCircle icon="arrows-clockwise" />}
      trailing={<MoneyText value={fromWire(payment.amount, currency)} size={14} weight="medium" />}
      trailingSubtitle={
        isDue ? t("recurring.dueNow") : t("recurring.dueOn", { date: shortDate(status.nextDueOn) })
      }
      onLongPress={() => confirmRemove(payment)}
      onPress={() =>
        update.mutate({ recurringId: payment.id, active: !payment.active })
      }
      divider={!last}
    />
    );
  };

  return (
    <Screen>
      <ScreenHeader
        gradient
        title={t("recurring.title")}
        subtitle={t("recurring.subtitle")}
        leading={{ icon: "list", label: t("nav.menu"), onPress: () => setDrawerOpen(true) }}
      />

      {payments.isPending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
        </View>
      ) : payments.isError ? (
        <View className="flex-1 justify-center gap-n3 px-n5">
          <Text className="text-[13.5px] leading-[21px] text-neutral-500">
            {toDisplayError(payments.error, t("common.loadFailed")).message}
          </Text>
          <Button title={t("common.tryAgain")} onPress={() => void payments.refetch()} />
        </View>
      ) : list.length === 0 ? (
        <View className="flex-1 justify-center">
          <EmptyState
            icon="arrows-clockwise"
            title={t("recurring.emptyTitle")}
            body={t("recurring.emptyBody")}
            action={{ label: t("recurring.new"), onPress: () => setCreating(true) }}
          />
        </View>
      ) : (
        <ScrollView className="flex-1 px-n4" contentContainerClassName="pb-[96px] pt-n4">
          {due.length > 0 ? (
            <ListSection title={t("recurring.dueSection")}>
              {due.map((status, index) => (
                <View key={status.payment?.id ?? index}>
                  {row(status, true, true)}
                  <View className="flex-row gap-n3 px-n4 pb-n3">
                    <Button
                      title={t("recurring.post")}
                      variant="ghost"
                      icon="check"
                      onPress={() =>
                        post.mutate({
                          recurringId: status.payment?.id ?? "",
                          dueOn: status.nextDueOn,
                        })
                      }
                    />
                    <Button
                      title={t("recurring.skip")}
                      variant="ghost"
                      icon="arrow-right"
                      onPress={() =>
                        skip.mutate({
                          recurringId: status.payment?.id ?? "",
                          dueOn: status.nextDueOn,
                        })
                      }
                    />
                  </View>
                  {index < due.length - 1 ? null : null}
                </View>
              ))}
            </ListSection>
          ) : null}

          {upcoming.length > 0 ? (
            <ListSection title={t("recurring.upcomingSection")} className="mt-n4">
              {upcoming.map((status, index) => row(status, false, index === upcoming.length - 1))}
            </ListSection>
          ) : null}

          <View className="mt-n4 overflow-hidden rounded-lg bg-surface">
            <ToggleRow
              label={t("recurring.showInactive")}
              value={showInactive}
              onValueChange={setShowInactive}
              divider={false}
            />
          </View>
          <Text className="mt-n3 px-n2 text-[11px] text-neutral-500">
            {t("recurring.tapHint")}
          </Text>
        </ScrollView>
      )}

      <Fab label={t("recurring.new")} onPress={() => setCreating(true)} />

      <NewRecurringSheet
        visible={creating}
        currency={currency}
        onClose={() => setCreating(false)}
      />

      <Drawer
        visible={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        account={{ name: family.data?.family?.name ?? "", email: "" }}
        household={{ name: family.data?.family?.name ?? "" }}
        items={drawerItems}
        activeId="recurring"
        onSelect={(item) => {
          setDrawerOpen(false);
          if (item.href && item.id !== "recurring") router.push(item.href);
        }}
      />
    </Screen>
  );
}

/** "щомісяця · наступний 5 вер" — the cadence in the household's own words. */
function cadenceLabel(t: Translate, payment: RecurringPayment, _currency: string): string {
  const interval = payment.cadence?.interval ?? 1;
  const unit = payment.cadence?.unit ?? RecurrenceUnit.MONTH;
  const key =
    unit === RecurrenceUnit.DAY
      ? "recurring.everyDay"
      : unit === RecurrenceUnit.WEEK
        ? "recurring.everyWeek"
        : unit === RecurrenceUnit.YEAR
          ? "recurring.everyYear"
          : "recurring.everyMonth";
  const cadence = t(key, { count: interval });
  return payment.active ? cadence : t("recurring.pausedWith", { cadence });
}

type Unit = "week" | "month" | "year";

function NewRecurringSheet({
  visible,
  currency,
  onClose,
}: {
  visible: boolean;
  currency: string;
  onClose: () => void;
}) {
  const { t } = useI18n();

  const [name, setName] = useState("");
  const [amountText, setAmountText] = useState("");
  const [unit, setUnit] = useState<Unit>("month");
  const [accountId, setAccountId] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [nextDueOn, setNextDueOn] = useState(() => toISODate(new Date()));
  const [autoPost, setAutoPost] = useState(false);
  const [picker, setPicker] = useState<"account" | "category" | null>(null);
  const [error, setError] = useState<string | null>(null);

  const accounts = useAccounts();
  const tree = useCategoryTree({ kind: TransactionKind.EXPENSE });
  const create = useCreateRecurringPayment();

  const selectable = useMemo(
    () => [...(accounts.data?.shared ?? []), ...(accounts.data?.privateOwn ?? [])],
    [accounts.data],
  );
  const account = selectable.find((a) => a.id === accountId) ?? selectable[0];
  const currencyCode = account?.currencyCode || currency;
  const amount: Money | null = parseAmount(amountText, currencyCode);
  const groups = tree.data ?? [];
  const category = groups.flatMap((node) => node.categories).find((c) => c.id === categoryId);

  const save = async () => {
    setError(null);
    if (name.trim() === "") {
      setError(t("recurring.nameRequired"));
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
    if (!isISODate(nextDueOn)) {
      setError(t("recurring.nextDueInvalid"));
      return;
    }
    try {
      await create.mutateAsync({
        name: name.trim(),
        amount,
        type: TransactionType.EXPENSE,
        categoryId,
        accountId: account.id,
        cadence: {
          interval: 1,
          unit:
            unit === "week"
              ? RecurrenceUnit.WEEK
              : unit === "year"
                ? RecurrenceUnit.YEAR
                : RecurrenceUnit.MONTH,
        },
        nextDueOn,
        autoPost,
      });
      setName("");
      setAmountText("");
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("recurring.saveError")).message);
    }
  };

  return (
    <>
      <Sheet visible={visible} onClose={onClose} title={t("recurring.new")} scroll>
        <View className="gap-n4">
          <Field label={t("recurring.name")} value={name} onChangeText={setName} />

          <AmountRow
            label={t("add.amount")}
            value={amountText}
            onChangeValue={setAmountText}
            currencyCode={currencyCode}
            invalid={error === t("add.amountRequired")}
          />

          <SegmentedTabs<Unit>
            value={unit}
            onChange={setUnit}
            options={[
              { value: "week", label: t("recurring.unitWeek") },
              { value: "month", label: t("recurring.unitMonth") },
              { value: "year", label: t("recurring.unitYear") },
            ]}
          />

          <Field
            label={t("recurring.nextDue")}
            value={nextDueOn}
            onChangeText={setNextDueOn}
            placeholder="2026-09-05"
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

          <View className="overflow-hidden rounded-lg bg-surface">
            <ToggleRow
              label={t("recurring.autoPost")}
              subtitle={t("recurring.autoPostHint")}
              value={autoPost}
              onValueChange={setAutoPost}
              divider={false}
            />
          </View>

          {error ? <Text className="text-[12.5px] text-overspend">{error}</Text> : null}

          <Button title={t("common.save")} onPress={() => void save()} disabled={create.isPending} />
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
