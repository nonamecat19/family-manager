import {
  fromWire,
  isISODate,
  parseAmount,
  RecurrenceUnit,
  SubscriptionStatus,
  TransactionType,
  toDisplayError,
  toInput,
  toISODate,
  useAccounts,
  useCancelSubscription,
  useCreateSubscription,
  useDeleteSubscription,
  useFinanceSettings,
  usePostSubscriptionOccurrence,
  useSubscriptions,
  useUpdateSubscription,
  type SubscriptionStatusView,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useEffect, useMemo, useState } from "react";
import { Alert, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  Card,
  Fab,
  formatMoney,
  IconCircle,
  ListSection,
  MoneyText,
  Row,
  ScrollSheet,
  shortDate,
  Stat,
  type Translate,
} from "@/components/kit";
import { AmountRow } from "@/components/screens/add-transaction/AmountRow";
import { AccountSheet } from "@/components/screens/add-transaction/pickers";
import { Button, Chip, EmptyState, Field, Kicker, Screen, ScreenHeader, ScrollBody, SettingsToggleRow } from "@fm/ui";

const FREQUENCY_PRESETS = [
  { value: "week", label: "recurring.unitWeek", interval: 1 },
  { value: "month", label: "recurring.unitMonth", interval: 1 },
  { value: "nmonth", label: "subscriptions.frequencyNMonthly", interval: 3 },
  { value: "year", label: "recurring.unitYear", interval: 1 },
] as const;

function addMonthsISO(iso: string, months: number): string {
  const [y, m, d] = iso.split("-").map(Number) as [number, number, number];
  const last = new Date(y, m - 1 + months + 1, 0).getDate();
  return toISODate(new Date(y, m - 1 + months, Math.min(d, last)));
}

function statusLine(t: Translate, sub: SubscriptionStatusView): string {
  const subscription = sub.subscription;
  if (!subscription) return "";
  if (subscription.status === SubscriptionStatus.CANCELLED) {
    return t("subscriptions.cancelled");
  }
  return t("subscriptions.dueOn", { date: shortDate(sub.nextDueOn) });
}

export default function SubscriptionsScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [showCancelled, setShowCancelled] = useState(false);
  const [creating, setCreating] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const settings = useFinanceSettings();
  const subscriptions = useSubscriptions(showCancelled);

  const currency = settings.data?.baseCurrencyCode || "UAH";
  const list = subscriptions.data ?? [];
  const active = list.filter((each) => each.subscription?.status === SubscriptionStatus.ACTIVE);
  const cancelled = list.filter((each) => each.subscription?.status !== SubscriptionStatus.ACTIVE);
  const selected = list.find((each) => each.subscription?.id === selectedId) ?? null;

  const inBase = active.filter((each) => (each.subscription?.amount?.currencyCode || currency) === currency);
  const monthlyTotal = inBase.reduce((sum, each) => sum + Number(each.monthlyApproximate?.amountMinor ?? 0), 0);
  const money = (minor: number) => ({ amountMinor: minor, currencyCode: currency });

  const row = (sub: SubscriptionStatusView, last: boolean) => {
    const subscription = sub.subscription;
    if (!subscription) return null;
    return (
      <View key={subscription.id}>
        <Row
          title={subscription.name}
          subtitle={statusLine(t, sub)}
          leading={<IconCircle icon="credit-card" />}
          trailing={<MoneyText value={fromWire(subscription.amount, currency)} size={14} weight="medium" />}
          trailingSubtitle={sub.monthlyApproximate ? t("subscriptions.monthlyApproximate", { amount: formatMoney(fromWire(sub.monthlyApproximate, currency)) }) : ""}
          onPress={() => setSelectedId(subscription.id)}
          divider={false}
        />
        {last ? null : <View className="ml-4.2 h-px w-full bg-divider" />}
      </View>
    );
  };

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader title={t("subscriptions.title")} onBack={() => router.back()} backLabel={t("common.back")} />

        {subscriptions.isPending ? (
          <View className="items-center py-7">
            <Text className="text-13 text-neutral-600">{t("common.loadingEllipsis")}</Text>
          </View>
        ) : subscriptions.isError ? (
          <View className="gap-3 py-4.5">
            <Text className="text-13.5 leading-[21px] text-neutral-600">
              {toDisplayError(subscriptions.error, t("common.loadFailed")).message}
            </Text>
            <Button title={t("common.tryAgain")} onPress={() => void subscriptions.refetch()} />
          </View>
        ) : list.length === 0 && !showCancelled ? (
          <EmptyState
            title={t("subscriptions.emptyTitle")}
            body={t("subscriptions.emptyBody")}
            action={{ label: t("subscriptions.new"), onPress: () => setCreating(true) }}
          />
        ) : (
          <>
            {active.length > 0 ? (
              <Card>
                <View className="flex-row gap-3">
                  <Stat label={t("subscriptions.monthlyTotal")} value={formatMoney(money(monthlyTotal))} />
                </View>
              </Card>
            ) : null}

            {active.length > 0 ? (
              <ListSection title={t("subscriptions.activeSection")}>
                {active.map((sub, index) => row(sub, index === active.length - 1))}
              </ListSection>
            ) : null}

            {showCancelled && cancelled.length > 0 ? (
              <ListSection title={t("subscriptions.cancelledSection")}>
                {cancelled.map((sub, index) => row(sub, index === cancelled.length - 1))}
              </ListSection>
            ) : null}

            <View className="overflow-hidden rounded-2xl bg-neutral-100">
              <SettingsToggleRow
                label={t("subscriptions.showCancelled")}
                value={showCancelled}
                onValueChange={setShowCancelled}
                divider={false}
              />
            </View>
            <Text className="text-11 text-neutral-600">{t("subscriptions.hint")}</Text>
          </>
        )}
      </ScrollBody>

      <Fab label={t("subscriptions.new")} onPress={() => setCreating(true)} />

      <NewSubscriptionSheet visible={creating} currency={currency} onClose={() => setCreating(false)} />
      <SubscriptionSheet
        subscription={selected}
        currency={currency}
        onClose={() => setSelectedId(null)}
      />
    </Screen>
  );
}

function NewSubscriptionSheet({
  visible,
  currency,
  onClose,
}: {
  visible: boolean;
  currency: string;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const today = useMemo(() => toISODate(new Date()), []);

  const [name, setName] = useState("");
  const [amountText, setAmountText] = useState("");
  const [frequency, setFrequency] = useState<"week" | "month" | "nmonth" | "year">("month");
  const [nMonths, setNMonths] = useState(3);
  const [dayOfMonth, setDayOfMonth] = useState(1);
  const [nextDueOn, setNextDueOn] = useState(() => addMonthsISO(today, 1));
  const [endOn, setEndOn] = useState("");
  const [autoPost, setAutoPost] = useState(false);
  const [pickingAccount, setPickingAccount] = useState(false);
  const [accountId, setAccountId] = useState("");
  const [error, setError] = useState<string | null>(null);

  const accounts = useAccounts();
  const create = useCreateSubscription();

  useEffect(() => {
    if (!visible) return;
    setName("");
    setAmountText("");
    setFrequency("month");
    setNMonths(3);
    setDayOfMonth(1);
    setNextDueOn(addMonthsISO(today, 1));
    setEndOn("");
    setAutoPost(false);
    setAccountId("");
    setError(null);
  }, [visible, today]);

  const selectable = useMemo(
    () => [...(accounts.data?.shared ?? []), ...(accounts.data?.privateOwn ?? [])].filter((a) => !a.archived),
    [accounts.data],
  );
  const account = selectable.find((each) => each.id === accountId) ?? selectable[0];
  const code = account?.currencyCode || currency;
  const amount = parseAmount(amountText, code);

  const save = async () => {
    setError(null);
    if (name.trim() === "") {
      setError(t("subscriptions.nameRequired"));
      return;
    }
    if (!amount || amount.amountMinor === 0) {
      setError(t("subscriptions.amountRequired"));
      return;
    }
    if (!isISODate(nextDueOn)) {
      setError(t("subscriptions.nextDueInvalid"));
      return;
    }
    if (!account) {
      setError(t("add.accountRequired"));
      return;
    }
    try {
      let unit: RecurrenceUnit;
      let interval = 1;
      if (frequency === "week") {
        unit = RecurrenceUnit.WEEK;
      } else if (frequency === "year") {
        unit = RecurrenceUnit.YEAR;
      } else if (frequency === "nmonth") {
        unit = RecurrenceUnit.MONTH;
        interval = nMonths;
      } else {
        unit = RecurrenceUnit.MONTH;
      }
      await create.mutateAsync({
        name: name.trim(),
        amount,
        type: TransactionType.EXPENSE,
        categoryId: "",
        accountId: account.id,
        cadence: { interval, unit, dayOfMonth },
        dayOfMonth,
        nextDueOn,
        endOn: endOn || undefined,
        autoPost,
      });
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("subscriptions.saveError")).message);
    }
  };

  return (
    <>
      <ScrollSheet visible={visible && !pickingAccount} onClose={onClose} title={t("subscriptions.new")}>
        <View className="gap-3 pt-1.4">
          <Field label={t("subscriptions.name")} value={name} onChangeText={setName} autoCapitalize="sentences" />

          <View className="gap-1.5">
            <Kicker>{t("subscriptions.amount")}</Kicker>
            <AmountRow label={t("subscriptions.amount")} value={amountText} onChangeValue={setAmountText} currencyCode={code} />
          </View>

          <View className="gap-1.5">
            <Kicker>{t("subscriptions.frequency")}</Kicker>
            <View className="flex-row flex-wrap gap-sm">
              {FREQUENCY_PRESETS.map((preset) => (
                <Chip
                  key={preset.value}
                  label={t(preset.label)}
                  active={frequency === preset.value}
                  onPress={() => {
                    setFrequency(preset.value as typeof frequency);
                    if (preset.value === "nmonth") setNMonths(preset.interval);
                  }}
                />
              ))}
            </View>
            {frequency === "nmonth" && (
              <View className="gap-1">
                <Kicker>{t("subscriptions.nMonths", { count: nMonths })}</Kicker>
                <Field
                  value={String(nMonths)}
                  onChangeText={(v) => setNMonths(Math.max(1, Math.min(12, parseInt(v, 10) || 1)))}
                  keyboardType="number-pad"
                  inputMode="numeric"
                />
              </View>
            )}
          </View>

          <View className="gap-1.5">
            <Kicker>{t("subscriptions.dayOfMonth")}</Kicker>
            <Field
              value={String(dayOfMonth)}
              onChangeText={(v) => setDayOfMonth(Math.max(1, Math.min(31, parseInt(v, 10) || 1)))}
              keyboardType="number-pad"
              inputMode="numeric"
            />
            <Text className="text-11.5 text-neutral-600">{t("subscriptions.dayOfMonthHint")}</Text>
          </View>

          <Field label={t("subscriptions.nextDue")} value={nextDueOn} onChangeText={setNextDueOn} />
          <Field label={t("subscriptions.endOn")} value={endOn} onChangeText={setEndOn} placeholder="2026-12-31" />
          <Text className="text-11.5 text-neutral-600">{t("subscriptions.autoPostHint")}</Text>

          <View className="overflow-hidden rounded-lg bg-surface">
            <SettingsToggleRow
              label={t("subscriptions.autoPost")}
              value={autoPost}
              onValueChange={setAutoPost}
              divider={false}
            />
          </View>

          <Button title={account ? account.name : t("add.account")} tone="quiet" onPress={() => setPickingAccount(true)} />

          {error ? <Text className="text-12.5 text-error">{error}</Text> : null}
          <Button title={t("common.save")} onPress={() => void save()} disabled={create.isPending} />
        </View>
      </ScrollSheet>

      <AccountSheet
        visible={visible && pickingAccount}
        onClose={() => setPickingAccount(false)}
        title={t("add.account")}
        shared={accounts.data?.shared ?? []}
        privateOwn={accounts.data?.privateOwn ?? []}
        sharedLabel={t("accounts.sharedSection")}
        privateLabel={t("accounts.privateSection", { name: "" })}
        selectedId={account?.id ?? ""}
        onSelect={(id) => {
          setAccountId(id);
          setPickingAccount(false);
        }}
      />
    </>
  );
}

function SubscriptionSheet({
  subscription,
  currency,
  onClose,
}: {
  subscription: SubscriptionStatusView | null;
  currency: string;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [name, setName] = useState("");
  const [amountText, setAmountText] = useState("");
  const [frequency, setFrequency] = useState<"week" | "month" | "nmonth" | "year">("month");
  const [nMonths, setNMonths] = useState(3);
  const [dayOfMonth, setDayOfMonth] = useState(1);
  const [nextDueOn, setNextDueOn] = useState("");
  const [endOn, setEndOn] = useState("");
  const [autoPost, setAutoPost] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const update = useUpdateSubscription();
  const post = usePostSubscriptionOccurrence();
  const cancel = useCancelSubscription();
  const remove = useDeleteSubscription();

  const sub = subscription?.subscription;
  const code = sub?.amount?.currencyCode || currency;

  useEffect(() => {
    if (!sub) return;
    setName(sub.name);
    setAmountText(toInput(fromWire(sub.amount, code)));
    const cadence = sub.cadence;
    if (cadence) {
      if (cadence.unit === RecurrenceUnit.WEEK) {
        setFrequency("week");
      } else if (cadence.unit === RecurrenceUnit.YEAR) {
        setFrequency("year");
      } else if (cadence.unit === RecurrenceUnit.MONTH && cadence.interval > 1) {
        setFrequency("nmonth");
        setNMonths(cadence.interval);
      } else {
        setFrequency("month");
      }
    }
    setDayOfMonth(sub.dayOfMonth || 1);
    setNextDueOn(sub.nextDueOn || "");
    setEndOn(sub.endOn || "");
    setAutoPost(sub.autoPost || false);
    setError(null);
  }, [sub, code]);

  if (!sub) {
    return <ScrollSheet visible={false} onClose={onClose} title="">{null}</ScrollSheet>;
  }

  const isActive = sub.status === SubscriptionStatus.ACTIVE;
  const busy = update.isPending || post.isPending || cancel.isPending || remove.isPending;

  const save = async () => {
    setError(null);
    if (name.trim() === "") {
      setError(t("subscriptions.nameRequired"));
      return;
    }
    const amount = parseAmount(amountText, code);
    if (!amount || amount.amountMinor === 0) {
      setError(t("subscriptions.amountRequired"));
      return;
    }
    if (!isISODate(nextDueOn)) {
      setError(t("subscriptions.nextDueInvalid"));
      return;
    }
    try {
      let unit: RecurrenceUnit;
      let interval = 1;
      if (frequency === "week") {
        unit = RecurrenceUnit.WEEK;
      } else if (frequency === "year") {
        unit = RecurrenceUnit.YEAR;
      } else if (frequency === "nmonth") {
        unit = RecurrenceUnit.MONTH;
        interval = nMonths;
      } else {
        unit = RecurrenceUnit.MONTH;
      }
      await update.mutateAsync({
        subscriptionId: sub.id,
        name: name.trim() !== sub.name ? name.trim() : undefined,
        amount: amount.amountMinor !== Number(sub.amount?.amountMinor ?? 0) ? amount : undefined,
        cadence: { interval, unit, dayOfMonth },
        dayOfMonth: dayOfMonth !== sub.dayOfMonth ? dayOfMonth : undefined,
        nextDueOn: nextDueOn !== sub.nextDueOn ? nextDueOn : undefined,
        endOn: endOn !== (sub.endOn || "") ? (endOn || undefined) : undefined,
        autoPost: autoPost !== sub.autoPost ? autoPost : undefined,
        active: sub.active,
      });
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("subscriptions.saveError")).message);
    }
  };

  const confirm = (title: string, body: string, action: () => void) =>
    Alert.alert(title, body, [
      { text: t("common.cancel"), style: "cancel" },
      { text: t("common.confirm"), style: "destructive", onPress: action },
    ]);

  return (
    <ScrollSheet visible onClose={onClose} title={sub.name}>
      <View className="gap-3.5 pt-1.4">
        <View className="flex-row gap-3">
          <Stat label={t("subscriptions.monthlyApproximate")} value={subscription.monthlyApproximate ? formatMoney(fromWire(subscription.monthlyApproximate, code)) : formatMoney({ amountMinor: 0, currencyCode: code })} />
        </View>
        <Text className="text-12 text-neutral-600">{statusLine(t, subscription)}</Text>

        {isActive ? (
          <>
            <Button
              title={t("subscriptions.payNow")}
              tone="quiet"
              disabled={busy}
              onPress={() => post.mutate(
                { subscriptionId: sub.id, dueOn: sub.nextDueOn },
                { onSuccess: onClose, onError: (e) => setError(toDisplayError(e, t("subscriptions.saveError")).message) },
              )}
            />
            <Field label={t("subscriptions.name")} value={name} onChangeText={setName} />
            <View className="gap-1.5">
              <Kicker>{t("subscriptions.amount")}</Kicker>
              <AmountRow label={t("subscriptions.amount")} value={amountText} onChangeValue={setAmountText} currencyCode={code} />
            </View>

            <View className="gap-1.5">
              <Kicker>{t("subscriptions.frequency")}</Kicker>
              <View className="flex-row flex-wrap gap-sm">
                {FREQUENCY_PRESETS.map((preset) => (
                  <Chip
                    key={preset.value}
                    label={t(preset.label)}
                    active={frequency === preset.value}
                    onPress={() => {
                      setFrequency(preset.value as typeof frequency);
                      if (preset.value === "nmonth") setNMonths(preset.interval);
                    }}
                  />
                ))}
              </View>
              {frequency === "nmonth" && (
                <View className="gap-1">
                  <Kicker>{t("subscriptions.nMonths", { count: nMonths })}</Kicker>
                  <Field
                    value={String(nMonths)}
                    onChangeText={(v) => setNMonths(Math.max(1, Math.min(12, parseInt(v, 10) || 1)))}
                    keyboardType="number-pad"
                    inputMode="numeric"
                  />
                </View>
              )}
            </View>

            <View className="gap-1.5">
              <Kicker>{t("subscriptions.dayOfMonth")}</Kicker>
              <Field
                value={String(dayOfMonth)}
                onChangeText={(v) => setDayOfMonth(Math.max(1, Math.min(31, parseInt(v, 10) || 1)))}
                keyboardType="number-pad"
                inputMode="numeric"
              />
            </View>

            <Field label={t("subscriptions.nextDue")} value={nextDueOn} onChangeText={setNextDueOn} />
            <Field label={t("subscriptions.endOn")} value={endOn} onChangeText={setEndOn} placeholder="2026-12-31" />

            <View className="overflow-hidden rounded-lg bg-surface">
              <SettingsToggleRow
                label={t("subscriptions.autoPost")}
                value={autoPost}
                onValueChange={setAutoPost}
                divider={false}
              />
            </View>

            {error ? <Text className="text-12.5 text-error">{error}</Text> : null}
            <Button title={t("common.save")} onPress={() => void save()} disabled={busy} />
          </>
        ) : error ? (
          <Text className="text-12.5 text-error">{error}</Text>
        ) : null}

        <View className="mt-sm gap-sm border-t border-divider pt-3">
          {isActive ? (
            <Button
              title={t("subscriptions.cancel")}
              tone="quiet"
              disabled={busy}
              onPress={() =>
                confirm(t("subscriptions.cancelTitle"), t("subscriptions.cancelBody", { name: sub.name }), () =>
                  cancel.mutate(sub.id, { onSuccess: onClose, onError: (e) => setError(toDisplayError(e, t("subscriptions.saveError")).message) }),
                )
              }
            />
          ) : null}
          <Button
            title={t("common.delete")}
            tone="danger"
            disabled={busy}
            onPress={() =>
              confirm(t("subscriptions.deleteTitle"), t("subscriptions.deleteBody", { name: sub.name }), () =>
                remove.mutate(sub.id, { onSuccess: onClose, onError: (e) => setError(toDisplayError(e, t("subscriptions.saveError")).message) }),
              )
            }
          />
        </View>
      </View>
    </ScrollSheet>
  );
}
