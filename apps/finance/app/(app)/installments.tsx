import {
  fromWire,
  InstallmentStatus,
  isISODate,
  parseAmount,
  toDisplayError,
  toInput,
  toISODate,
  useAccounts,
  useCancelInstallment,
  useCreateInstallment,
  useDeleteInstallment,
  useFinanceSettings,
  useInstallments,
  useUpdateInstallment,
  type Installment,
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

const MONTH_PRESETS = [3, 4, 6, 10, 12, 24] as const;

function addMonthsISO(iso: string, months: number): string {
  const [y, m, d] = iso.split("-").map(Number) as [number, number, number];
  const last = new Date(y, m - 1 + months + 1, 0).getDate();
  return toISODate(new Date(y, m - 1 + months, Math.min(d, last)));
}

function statusLine(t: Translate, inst: Installment): string {
  const progress = t("installments.progress", { made: inst.paymentsMade, months: inst.months });
  if (inst.status === InstallmentStatus.PAID_OFF) return `${progress} · ${t("installments.paidOff")}`;
  if (inst.status === InstallmentStatus.CANCELLED) return `${progress} · ${t("installments.cancelled")}`;
  return `${progress} · ${t("installments.nextOn", { date: shortDate(inst.nextDueOn) })}`;
}

function ratioPaid(inst: Installment): number {
  const total = Number(inst.total?.amountMinor ?? 0);
  return total > 0 ? Math.min(1, Number(inst.paid?.amountMinor ?? 0) / total) : 0;
}

function ProgressBar({ ratio }: { ratio: number }) {
  return (
    <View className="mx-[16.8px] mb-[10px] h-[5px] overflow-hidden rounded-full bg-neutral-200">
      <View className="h-full rounded-full bg-accent" style={{ width: `${Math.round(ratio * 100)}%` }} />
    </View>
  );
}

export default function InstallmentsScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [showClosed, setShowClosed] = useState(false);
  const [creating, setCreating] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const settings = useFinanceSettings();
  const installments = useInstallments(showClosed);

  const currency = settings.data?.baseCurrencyCode || "UAH";
  const list = installments.data ?? [];
  const active = list.filter((each) => each.status === InstallmentStatus.ACTIVE);
  const closed = list.filter((each) => each.status !== InstallmentStatus.ACTIVE);
  const selected = list.find((each) => each.id === selectedId) ?? null;

  const inBase = active.filter((each) => (each.remaining?.currencyCode || currency) === currency);
  const monthlyTotal = inBase.reduce((sum, each) => sum + Number(each.monthly?.amountMinor ?? 0), 0);
  const remainingTotal = inBase.reduce((sum, each) => sum + Number(each.remaining?.amountMinor ?? 0), 0);
  const money = (minor: number) => ({ amountMinor: minor, currencyCode: currency });

  const row = (inst: Installment, last: boolean) => (
    <View key={inst.id}>
      <Row
        title={inst.name}
        subtitle={statusLine(t, inst)}
        leading={<IconCircle icon="credit-card" />}
        trailing={<MoneyText value={fromWire(inst.remaining, currency)} size={14} weight="medium" />}
        trailingSubtitle={t("installments.perMonth", { amount: formatMoney(fromWire(inst.monthly, currency)) })}
        onPress={() => setSelectedId(inst.id)}
        divider={false}
      />
      <ProgressBar ratio={ratioPaid(inst)} />
      {last ? null : <View className="ml-[16.8px] h-px w-full bg-divider" />}
    </View>
  );

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader title={t("installments.title")} onBack={() => router.back()} backLabel={t("common.back")} />

        {installments.isPending ? (
          <View className="items-center py-[28px]">
            <Text className="text-[13px] text-neutral-600">{t("common.loadingEllipsis")}</Text>
          </View>
        ) : installments.isError ? (
          <View className="gap-[12px] py-[18px]">
            <Text className="text-[13.5px] leading-[21px] text-neutral-600">
              {toDisplayError(installments.error, t("common.loadFailed")).message}
            </Text>
            <Button title={t("common.tryAgain")} onPress={() => void installments.refetch()} />
          </View>
        ) : list.length === 0 && !showClosed ? (
          <EmptyState
            title={t("installments.emptyTitle")}
            body={t("installments.emptyBody")}
            action={{ label: t("installments.new"), onPress: () => setCreating(true) }}
          />
        ) : (
          <>
            {active.length > 0 ? (
              <Card>
                <View className="flex-row gap-[12px]">
                  <Stat label={t("installments.monthlyTotal")} value={formatMoney(money(monthlyTotal))} />
                  <Stat label={t("installments.remainingTotal")} value={formatMoney(money(remainingTotal))} />
                </View>
              </Card>
            ) : null}

            {active.length > 0 ? (
              <ListSection title={t("installments.activeSection")}>
                {active.map((inst, index) => row(inst, index === active.length - 1))}
              </ListSection>
            ) : null}

            {showClosed && closed.length > 0 ? (
              <ListSection title={t("installments.closedSection")}>
                {closed.map((inst, index) => row(inst, index === closed.length - 1))}
              </ListSection>
            ) : null}

            <View className="overflow-hidden rounded-2xl bg-neutral-100">
              <SettingsToggleRow
                label={t("installments.showClosed")}
                value={showClosed}
                onValueChange={setShowClosed}
                divider={false}
              />
            </View>
            <Text className="text-[11px] text-neutral-600">{t("installments.hint")}</Text>
          </>
        )}
      </ScrollBody>

      <Fab label={t("installments.new")} onPress={() => setCreating(true)} />

      <NewInstallmentSheet visible={creating} currency={currency} onClose={() => setCreating(false)} />
      <InstallmentSheet
        installment={selected}
        currency={currency}
        onClose={() => setSelectedId(null)}
        onPayExtra={(inst) => {
          setSelectedId(null);
          router.push({ pathname: "/(app)/add", params: { categoryId: inst.categoryId } });
        }}
      />
    </Screen>
  );
}

function NewInstallmentSheet({
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
  const [totalText, setTotalText] = useState("");
  const [monthsText, setMonthsText] = useState("12");
  const [monthlyText, setMonthlyText] = useState("");
  const [purchasedOn, setPurchasedOn] = useState(today);
  const [firstDueOn, setFirstDueOn] = useState(() => addMonthsISO(today, 1));
  const [accountId, setAccountId] = useState("");
  const [picking, setPicking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const accounts = useAccounts();
  const create = useCreateInstallment();

  useEffect(() => {
    if (!visible) return;
    setName("");
    setTotalText("");
    setMonthsText("12");
    setMonthlyText("");
    setPurchasedOn(today);
    setFirstDueOn(addMonthsISO(today, 1));
    setError(null);
  }, [visible, today]);

  const selectable = useMemo(
    () => [...(accounts.data?.shared ?? []), ...(accounts.data?.privateOwn ?? [])].filter((a) => !a.archived),
    [accounts.data],
  );
  const account = selectable.find((a) => a.id === accountId) ?? selectable[0];
  const code = account?.currencyCode || currency;
  const total = parseAmount(totalText, code);
  const months = Number.parseInt(monthsText, 10);
  const validMonths = Number.isFinite(months) && months >= 1 && months <= 120;
  const suggested =
    total && validMonths ? { amountMinor: Math.ceil(total.amountMinor / months), currencyCode: code } : null;
  const monthly = monthlyText.trim() === "" ? suggested : parseAmount(monthlyText, code);

  const save = async () => {
    setError(null);
    if (name.trim() === "") {
      setError(t("installments.nameRequired"));
      return;
    }
    if (!total || total.amountMinor === 0) {
      setError(t("add.amountRequired"));
      return;
    }
    if (!validMonths) {
      setError(t("installments.monthsInvalid"));
      return;
    }
    if (!monthly || monthly.amountMinor === 0) {
      setError(t("installments.monthlyInvalid"));
      return;
    }
    if (!isISODate(purchasedOn) || !isISODate(firstDueOn)) {
      setError(t("recurring.nextDueInvalid"));
      return;
    }
    if (!account) {
      setError(t("add.accountRequired"));
      return;
    }
    try {
      await create.mutateAsync({
        name: name.trim(),
        total,
        months,
        monthly: monthlyText.trim() === "" ? undefined : monthly,
        accountId: account.id,
        purchasedOn,
        firstDueOn,
      });
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("installments.saveError")).message);
    }
  };

  return (
    <>
      <ScrollSheet visible={visible && !picking} onClose={onClose} title={t("installments.new")}>
        <View className="gap-[12px] pt-[5.6px]">
          <Field label={t("installments.name")} value={name} onChangeText={setName} autoCapitalize="sentences" />

          <View className="gap-[6px]">
            <Kicker>{t("installments.total")}</Kicker>
            <AmountRow label={t("installments.total")} value={totalText} onChangeValue={setTotalText} currencyCode={code} />
          </View>

          <View className="gap-[8px]">
            <Kicker>{t("installments.months")}</Kicker>
            <View className="flex-row flex-wrap gap-[8px]">
              {MONTH_PRESETS.map((preset) => (
                <Chip
                  key={preset}
                  label={String(preset)}
                  active={months === preset}
                  onPress={() => setMonthsText(String(preset))}
                />
              ))}
            </View>
            <Field value={monthsText} onChangeText={setMonthsText} keyboardType="number-pad" inputMode="numeric" />
          </View>

          <Field
            label={t("installments.monthly")}
            value={monthlyText}
            onChangeText={setMonthlyText}
            keyboardType="decimal-pad"
            inputMode="decimal"
            placeholder={suggested ? toInput(suggested) : "0"}
          />

          <Field label={t("installments.purchasedOn")} value={purchasedOn} onChangeText={setPurchasedOn} />
          <Field label={t("installments.firstDueOn")} value={firstDueOn} onChangeText={setFirstDueOn} />
          <Text className="text-[11.5px] text-neutral-600">{t("installments.firstDueHint")}</Text>

          <Button title={account ? account.name : t("add.account")} tone="quiet" onPress={() => setPicking(true)} />

          {error ? <Text className="text-[12.5px] text-error">{error}</Text> : null}
          <Button title={t("common.save")} onPress={() => void save()} disabled={create.isPending} />
        </View>
      </ScrollSheet>

      <AccountSheet
        visible={visible && picking}
        onClose={() => setPicking(false)}
        title={t("add.account")}
        shared={accounts.data?.shared ?? []}
        privateOwn={accounts.data?.privateOwn ?? []}
        sharedLabel={t("accounts.sharedSection")}
        privateLabel={t("accounts.privateSection", { name: "" })}
        selectedId={account?.id ?? ""}
        onSelect={(id) => {
          setAccountId(id);
          setPicking(false);
        }}
      />
    </>
  );
}

function InstallmentSheet({
  installment,
  currency,
  onClose,
  onPayExtra,
}: {
  installment: Installment | null;
  currency: string;
  onClose: () => void;
  onPayExtra: (inst: Installment) => void;
}) {
  const { t } = useI18n();
  const [name, setName] = useState("");
  const [monthlyText, setMonthlyText] = useState("");
  const [nextDueOn, setNextDueOn] = useState("");
  const [error, setError] = useState<string | null>(null);

  const update = useUpdateInstallment();
  const cancel = useCancelInstallment();
  const remove = useDeleteInstallment();

  const code = installment?.total?.currencyCode || currency;

  useEffect(() => {
    if (!installment) return;
    setName(installment.name);
    setMonthlyText(toInput(fromWire(installment.monthly, code)));
    setNextDueOn(installment.nextDueOn);
    setError(null);
  }, [installment, code]);

  if (!installment) {
    return <ScrollSheet visible={false} onClose={onClose} title="">{null}</ScrollSheet>;
  }

  const isActive = installment.status === InstallmentStatus.ACTIVE;
  const busy = update.isPending || cancel.isPending || remove.isPending;
  const fail = (e: unknown) => setError(toDisplayError(e, t("installments.saveError")).message);

  const save = async () => {
    setError(null);
    const monthly = parseAmount(monthlyText, code);
    if (!monthly || monthly.amountMinor === 0) {
      setError(t("installments.monthlyInvalid"));
      return;
    }
    if (!isISODate(nextDueOn)) {
      setError(t("recurring.nextDueInvalid"));
      return;
    }
    if (name.trim() === "") {
      setError(t("installments.nameRequired"));
      return;
    }
    try {
      await update.mutateAsync({
        installmentId: installment.id,
        name: name.trim() !== installment.name ? name.trim() : undefined,
        monthly: monthly.amountMinor !== Number(installment.monthly?.amountMinor ?? 0) ? monthly : undefined,
        nextDueOn: nextDueOn !== installment.nextDueOn ? nextDueOn : undefined,
      });
      onClose();
    } catch (e) {
      fail(e);
    }
  };

  const confirm = (title: string, body: string, action: () => void) =>
    Alert.alert(title, body, [
      { text: t("common.cancel"), style: "cancel" },
      { text: t("common.confirm"), style: "destructive", onPress: action },
    ]);

  return (
    <ScrollSheet visible onClose={onClose} title={installment.name}>
      <View className="gap-[14px] pt-[5.6px]">
        <View className="flex-row gap-[12px]">
          <Stat label={t("installments.paid")} value={formatMoney(fromWire(installment.paid, code))} />
          <Stat label={t("installments.remaining")} value={formatMoney(fromWire(installment.remaining, code))} />
          <Stat label={t("installments.totalShort")} value={formatMoney(fromWire(installment.total, code))} />
        </View>
        <Text className="text-[12px] text-neutral-600">{statusLine(t, installment)}</Text>

        {isActive ? (
          <>
            <Button title={t("installments.payExtra")} tone="quiet" onPress={() => onPayExtra(installment)} />
            <Field label={t("installments.name")} value={name} onChangeText={setName} />
            <Field
              label={t("installments.monthly")}
              value={monthlyText}
              onChangeText={setMonthlyText}
              keyboardType="decimal-pad"
              inputMode="decimal"
            />
            <Field label={t("installments.nextDue")} value={nextDueOn} onChangeText={setNextDueOn} />
            {error ? <Text className="text-[12.5px] text-error">{error}</Text> : null}
            <Button title={t("common.save")} onPress={() => void save()} disabled={busy} />
          </>
        ) : error ? (
          <Text className="text-[12.5px] text-error">{error}</Text>
        ) : null}

        <View className="mt-[8px] gap-[8px] border-t border-divider pt-[12px]">
          {isActive ? (
            <Button
              title={t("installments.cancel")}
              tone="quiet"
              disabled={busy}
              onPress={() =>
                confirm(t("installments.cancelTitle"), t("installments.cancelBody", { name: installment.name }), () =>
                  cancel.mutate(installment.id, { onSuccess: onClose, onError: fail }),
                )
              }
            />
          ) : null}
          {installment.paymentsMade === 0 ? (
            <Button
              title={t("common.delete")}
              tone="danger"
              disabled={busy}
              onPress={() =>
                confirm(t("installments.deleteTitle"), t("installments.deleteBody", { name: installment.name }), () =>
                  remove.mutate(installment.id, { onSuccess: onClose, onError: fail }),
                )
              }
            />
          ) : null}
        </View>
      </View>
    </ScrollSheet>
  );
}
