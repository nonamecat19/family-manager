import {
  accountScope,
  currentPeriod,
  familyScope,
  fromWire,
  memberScope,
  periodWindow,
  stepPeriod,
  toDisplayError,
  toISODate,
  TransactionKind,
  useFamily,
  useHomeSummary,
  useLogTemplate,
  type PeriodInput,
  type ScopeInput,
  type SteppablePeriod,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { ActivityIndicator, ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { AmountChip, Card, dayHeading, DonutChart, type DonutSegment, Fab, formatMoney, monthTitle, PeriodStepper, type PeriodTab, PeriodTabs, type Scope, ScopeSwitcher, SegmentedTabs, seriesColor, shortDate, type Translate } from "@/components/kit";
import { HomeGroupCard } from "@/components/screens/home/groupCard.tsx";
import { Button, EmptyState, IconButton, iconOr, Kicker, organic, Screen, ScreenHeader } from "@fm/ui";

export default function HomeScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [scope, setScope] = useState<Scope>({ kind: "family" });
  const [tab, setTab] = useState<PeriodTab>("month");
  const [period, setPeriod] = useState<PeriodInput>(() => currentPeriod("month"));
  const [kind, setKind] = useState<"expense" | "income">("expense");

  const apiScope: ScopeInput =
    scope.kind === "family"
      ? familyScope
      : scope.kind === "member"
        ? memberScope(scope.id)
        : accountScope(scope.id);

  const summary = useHomeSummary({
    scope: apiScope,
    period,
    kind: kind === "expense" ? TransactionKind.EXPENSE : TransactionKind.INCOME,
  });
  const family = useFamily();
  const logTemplate = useLogTemplate();

  const today = toISODate(new Date());
  const window = periodWindow(period);
  const data = summary.data;
  const currency = data?.headlineBalance?.currencyCode || data?.periodTotal?.currencyCode || "UAH";

  const members = useMemo(
    () => (data?.members ?? []).map((m) => ({ id: m.memberId, name: m.displayName })),
    [data?.members],
  );

  const segments: DonutSegment[] = useMemo(
    () =>
      (data?.slices ?? []).map((slice) => ({
        id: slice.groupId,
        label: slice.name || t("home.uncategorised"),
        value: fromWire(slice.amount, currency).amountMinor,
        color: seriesColor(slice.colorStep),
      })),
    [data?.slices, currency, t],
  );

  const templateOwner =
    scope.kind === "member"
      ? (members.find((m) => m.id === scope.id)?.name ?? family.data?.family?.name ?? "")
      : (family.data?.family?.name ?? "");

  const step = (by: number) => {
    if (period.granularity === "custom") return;
    setPeriod(stepPeriod(period as SteppablePeriod, by));
  };

  const onTab = (next: PeriodTab) => {
    setTab(next);
    if (next === "custom") {
      setPeriod({ granularity: "custom", range: periodWindow(period) });
      return;
    }
    const anchor = period.granularity === "custom" ? period.range.from : period.anchor;
    setPeriod({ granularity: next, anchor });
  };

  const openGroup = (groupId: string) =>
    router.push({
      pathname: "/(app)/transactions",
      params: {
        groupId,
        kind,
        granularity: period.granularity,
        anchor: period.granularity === "custom" ? period.range.from : period.anchor,
      },
    });

  const header = (
    <View className="gap-lg px-5.5 pt-sm">
      <ScreenHeader
        title={t("home.title")}
        actions={
          <IconButton
            icon="receipt"
            label={t("transactions.title")}
            onPress={() => router.push("/(app)/transactions")}
          />
        }
      />
      <ScopeSwitcher
        scope={scope}
        members={members}
        onChange={setScope}
        balance={fromWire(data?.headlineBalance, currency)}
      />
      <SegmentedTabs
        options={[
          { value: "expense" as const, label: t("common.expenses") },
          { value: "income" as const, label: t("common.income") },
        ]}
        value={kind}
        onChange={setKind}
      />
    </View>
  );

  if (summary.isPending) {
    return (
      <Screen>
        {header}
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator color={organic.accent.DEFAULT} />
        </View>
      </Screen>
    );
  }

  if (summary.isError) {
    const shown = toDisplayError(summary.error, t("common.loadFailed"));
    return (
      <Screen>
        {header}
        <View className="flex-1 justify-center gap-2.8 px-5.6">
          <Text className="text-13.5 leading-[21px] text-neutral-600">{shown.message}</Text>
          {shown.reference ? (
            <Text className="text-12 text-neutral-600">
              {t("common.errorReference", { ref: shown.reference })}
            </Text>
          ) : null}
          <Button title={t("common.tryAgain")} onPress={() => void summary.refetch()} />
        </View>
      </Screen>
    );
  }

  const groups = data?.groups ?? [];
  const templates = data?.templates ?? [];
  const periodTotal = fromWire(data?.periodTotal, currency);

  return (
    <Screen>
      {header}

      <ScrollView
        className="flex-1"
        contentContainerStyle={{ padding: 11.2, paddingBottom: 96, gap: 8.4 }}
        showsVerticalScrollIndicator={false}
      >
        <Card>
          <PeriodTabs value={tab} onChange={onTab} />
          <PeriodStepper
            label={periodLabel(t, period)}
            onPrev={() => step(-1)}
            onNext={() => step(1)}
            nextDisabled={period.granularity === "custom" || window.to >= today}
            className="mt-2.8"
          />
          <View className="mt-2.8 items-center">
            <DonutChart
              segments={segments}
              size={178}
              thickness={34}
              centerValue={formatMoney(periodTotal)}
              centerLabel={t("common.groupCount", {
                count: data?.groupCount ?? groups.filter((g) => g.groupId !== "").length,
              })}
              onPressSegment={(segment) => {
                if (segment.id !== "") openGroup(segment.id);
              }}
            />
          </View>
        </Card>

        {templates.length > 0 ? (
          <View>
            <Kicker className="mb-2.1 ml-0.7">
              {t("home.quickTemplates", { name: templateOwner })}
            </Kicker>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={{ gap: 5.6 }}
            >
              {templates.map((template) => (
                <AmountChip
                  key={template.id}
                  label={template.label}
                  icon={iconOr(template.icon)}
                  amount={fromWire(template.amount, currency)}
                  onPress={() => logTemplate.mutate({ templateId: template.id })}
                  onLongPress={() => router.push(`/(app)/add?templateId=${template.id}`)}
                />
              ))}
            </ScrollView>
          </View>
        ) : null}

        {groups.length === 0 ? (
          <EmptyState
            title={t("home.emptyTitle")}
            body={t("home.emptyBody")}
            action={{ label: t("home.addTransaction"), onPress: () => router.push("/(app)/add") }}
          />
        ) : (
          <View className="gap-1.4">
            {groups.map((group) => {
              const status = group.budget;
              const limit = status?.budget?.limit ? fromWire(status.budget.limit, currency) : null;
              const spent = status ? fromWire(status.spent, currency) : null;
              const amount = fromWire(group.amount, currency);
              const contributors = (group.contributorMemberIds ?? [])
                .map((id) => members.find((m) => m.id === id)?.name)
                .filter((name): name is string => Boolean(name));
              const categories = t("common.categoryCount", { count: group.categoryCount });

              return (
                <HomeGroupCard
                  key={group.groupId || "uncategorised"}
                  name={group.name || t("home.uncategorised")}
                  icon={group.icon}
                  colorStep={group.colorStep}
                  amount={amount}
                  meta={
                    contributors.length > 0
                      ? t("home.groupMeta", { categories, name: contributors.join(", ") })
                      : categories
                  }
                  caption={
                    limit
                      ? t("common.ofLimit", { amount: formatMoney(limit) })
                      : t("common.percent", { value: Math.round(group.share * 100) })
                  }
                  budget={
                    limit && spent
                      ? {
                          spentMinor: spent.amountMinor,
                          limitMinor: limit.amountMinor,
                          over: status?.exceeded ?? false,
                        }
                      : undefined
                  }
                  onPress={group.groupId === "" ? undefined : () => openGroup(group.groupId)}
                />
              );
            })}
          </View>
        )}
      </ScrollView>

      <Fab label={t("home.addTransaction")} onPress={() => router.push("/(app)/add")} />
    </Screen>
  );
}

function periodLabel(t: Translate, period: PeriodInput): string {
  if (period.granularity === "custom") {
    return `${shortDate(period.range.from)} – ${shortDate(period.range.to)}`;
  }
  switch (period.granularity) {
    case "day":
      return dayHeading(t, period.anchor);
    case "week": {
      const window = periodWindow(period);
      return `${shortDate(window.from)} – ${shortDate(window.to)}`;
    }
    case "month": {
      const [year, month] = period.anchor.split("-");
      return monthTitle(t, Number(year), Number(month));
    }
    case "year":
      return period.anchor.slice(0, 4);
  }
}
