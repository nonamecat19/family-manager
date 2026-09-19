import {
  fromWire,
  InvestmentKind,
  parseAmount,
  toDisplayError,
  toInput,
  useCreateInvestment,
  useDeleteInvestment,
  useFinanceSettings,
  useInvestments,
  useSetInvestmentValue,
  useUpdateInvestment,
  type Investment,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { Alert, Text, View } from "react-native";

import { useI18n, type TranslationKey } from "@/components/i18n";
import {
  Card,
  Fab,
  formatMoney,
  formatPercent,
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
import { Button, Chip, EmptyState, Field, Kicker, Screen, ScreenHeader, ScrollBody, SettingsToggleRow } from "@fm/ui";

const KINDS: readonly { kind: InvestmentKind; icon: string; key: TranslationKey }[] = [
  { kind: InvestmentKind.DEPOSIT, icon: "piggy-bank", key: "investments.kindDeposit" },
  { kind: InvestmentKind.STOCKS, icon: "trend-up", key: "investments.kindStocks" },
  { kind: InvestmentKind.BONDS, icon: "chart-bar", key: "investments.kindBonds" },
  { kind: InvestmentKind.CRYPTO, icon: "currency-btc", key: "investments.kindCrypto" },
  { kind: InvestmentKind.REAL_ESTATE, icon: "house-line", key: "investments.kindRealEstate" },
  { kind: InvestmentKind.OTHER, icon: "chart-donut", key: "investments.kindOther" },
];

function kindMeta(kind: InvestmentKind) {
  return KINDS.find((each) => each.kind === kind) ?? KINDS[KINDS.length - 1]!;
}

function kindLabel(t: Translate, kind: InvestmentKind): string {
  return t(kindMeta(kind).key);
}

export default function InvestmentsScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [showArchived, setShowArchived] = useState(false);
  const [creating, setCreating] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const settings = useFinanceSettings();
  const investments = useInvestments(showArchived);

  const currency = settings.data?.baseCurrencyCode || "UAH";
  const list = investments.data ?? [];
  const selected = list.find((each) => each.id === selectedId) ?? null;

  const inBase = list.filter((each) => !each.archived && (each.invested?.currencyCode || currency) === currency);
  const investedTotal = inBase.reduce((sum, each) => sum + Number(each.invested?.amountMinor ?? 0), 0);
  const valueTotal = inBase.reduce((sum, each) => sum + Number(each.currentValue?.amountMinor ?? 0), 0);
  const profitTotal = valueTotal - investedTotal;
  const money = (minor: number) => ({ amountMinor: minor, currencyCode: currency });

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader title={t("investments.title")} onBack={() => router.back()} backLabel={t("common.back")} />

        {investments.isPending ? (
          <View className="items-center py-[28px]">
            <Text className="text-[13px] text-neutral-600">{t("common.loadingEllipsis")}</Text>
          </View>
        ) : investments.isError ? (
          <View className="gap-3 py-[18px]">
            <Text className="text-[13.5px] leading-[21px] text-neutral-600">
              {toDisplayError(investments.error, t("common.loadFailed")).message}
            </Text>
            <Button title={t("common.tryAgain")} onPress={() => void investments.refetch()} />
          </View>
        ) : list.length === 0 ? (
          <EmptyState
            title={t("investments.emptyTitle")}
            body={t("investments.emptyBody")}
            action={{ label: t("investments.new"), onPress: () => setCreating(true) }}
          />
        ) : (
          <>
            <Card>
              <View className="flex-row gap-3">
                <Stat label={t("investments.invested")} value={formatMoney(money(investedTotal))} />
                <Stat label={t("investments.value")} value={formatMoney(money(valueTotal))} />
                <Stat
                  label={t("investments.profit")}
                  value={
                    investedTotal > 0
                      ? `${formatMoney(money(profitTotal))} · ${formatPercent(profitTotal / investedTotal)}`
                      : formatMoney(money(profitTotal))
                  }
                  tone={profitTotal < 0 ? "overspend" : profitTotal > 0 ? "positive" : "default"}
                />
              </View>
            </Card>

            <ListSection title={t("investments.portfolio")}>
              {list.map((investment, index) => {
                const profit = Number(investment.profit?.amountMinor ?? 0);
                return (
                  <Row
                    key={investment.id}
                    title={investment.name}
                    subtitle={
                      investment.valueUpdatedOn
                        ? t("investments.rowMeta", {
                            kind: kindLabel(t, investment.kind),
                            date: shortDate(investment.valueUpdatedOn),
                          })
                        : kindLabel(t, investment.kind)
                    }
                    leading={<IconCircle icon={kindMeta(investment.kind).icon} index={index} />}
                    trailing={
                      <MoneyText value={fromWire(investment.currentValue, currency)} size={14} weight="medium" />
                    }
                    trailingSubtitle={
                      Number(investment.invested?.amountMinor ?? 0) > 0
                        ? `${profit >= 0 ? "+" : ""}${formatPercent(investment.profitBps / 10000)}`
                        : t("investments.noContributions")
                    }
                    onPress={() => setSelectedId(investment.id)}
                    divider={index < list.length - 1}
                  />
                );
              })}
            </ListSection>

            <View className="overflow-hidden rounded-2xl bg-neutral-100">
              <SettingsToggleRow
                label={t("investments.showArchived")}
                value={showArchived}
                onValueChange={setShowArchived}
                divider={false}
              />
            </View>
            <Text className="text-[11px] text-neutral-600">{t("investments.hint")}</Text>
          </>
        )}
      </ScrollBody>

      <Fab label={t("investments.new")} onPress={() => setCreating(true)} />

      <NewInvestmentSheet visible={creating} onClose={() => setCreating(false)} />
      <InvestmentSheet
        investment={selected}
        currency={currency}
        onClose={() => setSelectedId(null)}
        onContribute={(investment) => {
          setSelectedId(null);
          router.push({ pathname: "/(app)/add", params: { categoryId: investment.categoryId } });
        }}
      />
    </Screen>
  );
}

function KindChips({ value, onChange }: { value: InvestmentKind; onChange: (next: InvestmentKind) => void }) {
  const { t } = useI18n();
  return (
    <View className="flex-row flex-wrap gap-[8px]">
      {KINDS.map((each) => (
        <Chip
          key={each.kind}
          label={kindLabel(t, each.kind)}
          active={value === each.kind}
          onPress={() => onChange(each.kind)}
        />
      ))}
    </View>
  );
}

function NewInvestmentSheet({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const { t } = useI18n();
  const [name, setName] = useState("");
  const [kind, setKind] = useState<InvestmentKind>(InvestmentKind.STOCKS);
  const [error, setError] = useState<string | null>(null);
  const create = useCreateInvestment();

  useEffect(() => {
    if (!visible) return;
    setName("");
    setKind(InvestmentKind.STOCKS);
    setError(null);
  }, [visible]);

  const save = async () => {
    setError(null);
    if (name.trim() === "") {
      setError(t("investments.nameRequired"));
      return;
    }
    try {
      await create.mutateAsync({ name: name.trim(), kind });
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("investments.saveError")).message);
    }
  };

  return (
    <ScrollSheet visible={visible} onClose={onClose} title={t("investments.new")}>
      <View className="gap-3.5 pt-1.4">
        <Field label={t("investments.name")} value={name} onChangeText={setName} autoCapitalize="sentences" />
        <View className="gap-[8px]">
          <Kicker>{t("investments.kind")}</Kicker>
          <KindChips value={kind} onChange={setKind} />
        </View>
        <Text className="text-[11.5px] text-neutral-600">{t("investments.createHint")}</Text>
        {error ? <Text className="text-[12.5px] text-error">{error}</Text> : null}
        <Button title={t("common.save")} onPress={() => void save()} disabled={create.isPending} />
      </View>
    </ScrollSheet>
  );
}

function InvestmentSheet({
  investment,
  currency,
  onClose,
  onContribute,
}: {
  investment: Investment | null;
  currency: string;
  onClose: () => void;
  onContribute: (investment: Investment) => void;
}) {
  const { t } = useI18n();
  const [valueText, setValueText] = useState("");
  const [name, setName] = useState("");
  const [kind, setKind] = useState<InvestmentKind>(InvestmentKind.OTHER);
  const [error, setError] = useState<string | null>(null);

  const setValue = useSetInvestmentValue();
  const update = useUpdateInvestment();
  const remove = useDeleteInvestment();

  const code = investment?.currentValue?.currencyCode || currency;

  useEffect(() => {
    if (!investment) return;
    setValueText(toInput(fromWire(investment.currentValue, code)));
    setName(investment.name);
    setKind(investment.kind);
    setError(null);
  }, [investment, code]);

  if (!investment) {
    return <ScrollSheet visible={false} onClose={onClose} title="">{null}</ScrollSheet>;
  }

  const busy = setValue.isPending || update.isPending || remove.isPending;
  const fail = (e: unknown) => setError(toDisplayError(e, t("investments.saveError")).message);

  const save = async () => {
    setError(null);
    const value = parseAmount(valueText, code);
    if (!value) {
      setError(t("add.amountRequired"));
      return;
    }
    if (name.trim() === "") {
      setError(t("investments.nameRequired"));
      return;
    }
    try {
      if (name.trim() !== investment.name || kind !== investment.kind) {
        await update.mutateAsync({ investmentId: investment.id, name: name.trim(), kind });
      }
      if (value.amountMinor !== Number(investment.currentValue?.amountMinor ?? 0) || !investment.valueUpdatedOn) {
        await setValue.mutateAsync({ investmentId: investment.id, value });
      }
      onClose();
    } catch (e) {
      fail(e);
    }
  };

  const confirmDelete = () => {
    Alert.alert(t("investments.deleteTitle"), t("investments.deleteBody", { name: investment.name }), [
      { text: t("common.cancel"), style: "cancel" },
      {
        text: t("common.delete"),
        style: "destructive",
        onPress: () => remove.mutate(investment.id, { onSuccess: onClose, onError: fail }),
      },
    ]);
  };

  return (
    <ScrollSheet visible onClose={onClose} title={investment.name}>
      <View className="gap-3.5 pt-1.4">
        <View className="flex-row gap-3">
          <Stat label={t("investments.invested")} value={formatMoney(fromWire(investment.invested, code))} />
          <Stat
            label={t("investments.profit")}
            value={formatMoney(fromWire(investment.profit, code))}
            tone={Number(investment.profit?.amountMinor ?? 0) < 0 ? "overspend" : "positive"}
          />
        </View>

        <Button title={t("investments.contribute")} onPress={() => onContribute(investment)} />

        <View className="gap-[6px]">
          <Kicker>{t("investments.currentValue")}</Kicker>
          <AmountRow label={t("investments.currentValue")} value={valueText} onChangeValue={setValueText} currencyCode={code} />
        </View>

        <Field label={t("investments.name")} value={name} onChangeText={setName} />
        <KindChips value={kind} onChange={setKind} />

        {error ? <Text className="text-[12.5px] text-error">{error}</Text> : null}
        <Button title={t("common.save")} onPress={() => void save()} disabled={busy} />

        <View className="mt-[8px] gap-[8px] border-t border-divider pt-3">
          <Button
            title={investment.archived ? t("investments.unarchive") : t("investments.archive")}
            tone="quiet"
            disabled={busy}
            onPress={() =>
              update.mutate(
                { investmentId: investment.id, archived: !investment.archived },
                { onSuccess: onClose, onError: fail },
              )
            }
          />
          <Button title={t("common.delete")} tone="danger" onPress={confirmDelete} disabled={busy} />
        </View>
      </View>
    </ScrollSheet>
  );
}
