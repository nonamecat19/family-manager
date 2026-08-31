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
import {
  Button,
  Card,
  Chip,
  dayHeading,
  DonutChart,
  Drawer,
  EmptyState,
  Fab,
  formatMoney,
  iconOr,
  Kicker,
  monthTitle,
  nocturne,
  PeriodTabs,
  PeriodStepper,
  Screen,
  ScreenHeader,
  ScopeSwitcher,
  SegmentedTabs,
  seriesColor,
  shortDate,
  useDrawerItems,
  type DonutSegment,
  type PeriodTab,
  type Scope,
  type Translate,
} from "@/components/nocturne";
import { HomeGroupCard } from "@/components/screens/home/groupCard.tsx";

/**
 * Screen 02 — Home. The whole screen is one call: `GetHomeSummary` returns the headline
 * balance, the period total, the donut slices, the group rows, the member chips and the
 * caller's quick templates for exactly the scope/period/kind the controls are set to. Nothing
 * here recomputes a total client-side, so what the donut says and what the rows add up to
 * cannot drift, and private accounts stay out of the family view because the server leaves
 * them out — the app never filters them itself.
 */
export default function HomeScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [scope, setScope] = useState<Scope>({ kind: "family" });
  const [tab, setTab] = useState<PeriodTab>("month");
  const [period, setPeriod] = useState<PeriodInput>(() => currentPeriod("month"));
  const [kind, setKind] = useState<"expense" | "income">("expense");
  const [menuOpen, setMenuOpen] = useState(false);

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
  const drawerItems = useDrawerItems();

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

  // The design's "Швидко · шаблони Сергія". The templates are the caller's own (the server
  // defaults ListTemplates/GetHomeSummary to the caller), but no RPC tells the app who the
  // caller is — see the note in the return value. Naming the selected member when there is
  // one, and the household otherwise, is the closest honest label.
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
      // No date-range picker exists in the kit yet; "Період" opens on the window currently
      // shown rather than on an empty range.
      setPeriod({ granularity: "custom", range: periodWindow(period) });
      return;
    }
    const anchor = period.granularity === "custom" ? period.range.from : period.anchor;
    setPeriod({ granularity: next, anchor });
  };

  const openGroup = (groupId: string) =>
    router.push({
      pathname: "/(app)/transactions",
      // Both the tab and the window travel with the tap: a group row read under "Доходи" in
      // July must open the July income feed, not the current month's expenses.
      params: {
        groupId,
        kind,
        granularity: period.granularity,
        anchor: period.granularity === "custom" ? period.range.from : period.anchor,
      },
    });

  const header = (
    <ScreenHeader
      title={t("home.title")}
      gradient
      leading={{ icon: "list", label: t("nav.menu"), onPress: () => setMenuOpen(true) }}
      trailing={{
        icon: "receipt",
        label: t("transactions.title"),
        onPress: () => router.push("/(app)/transactions"),
      }}
    >
      <ScopeSwitcher
        scope={scope}
        members={members}
        onChange={setScope}
        balance={fromWire(data?.headlineBalance, currency)}
        className="mt-n3"
      />
      <SegmentedTabs
        options={[
          { value: "expense" as const, label: t("common.expenses") },
          { value: "income" as const, label: t("common.income") },
        ]}
        value={kind}
        onChange={setKind}
        className="mt-n4"
      />
    </ScreenHeader>
  );

  if (summary.isPending) {
    return (
      <Screen>
        {header}
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator color={nocturne.accent.DEFAULT} />
        </View>
      </Screen>
    );
  }

  if (summary.isError) {
    const shown = toDisplayError(summary.error, t("common.loadFailed"));
    return (
      <Screen>
        {header}
        <View className="flex-1 justify-center gap-n4 px-n6">
          <Text className="text-[13.5px] leading-[21px] text-neutral-500">{shown.message}</Text>
          {shown.reference ? (
            <Text className="text-[12px] text-neutral-600">
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
        contentContainerStyle={{ padding: nocturne.space.n4, paddingBottom: 96, gap: nocturne.space.n3 }}
        showsVerticalScrollIndicator={false}
      >
        <Card>
          <PeriodTabs value={tab} onChange={onTab} />
          <PeriodStepper
            label={periodLabel(t, period)}
            onPrev={() => step(-1)}
            onNext={() => step(1)}
            nextDisabled={period.granularity === "custom" || window.to >= today}
            className="mt-n4"
          />
          <View className="mt-n4 items-center">
            <DonutChart
              segments={segments}
              size={178}
              thickness={34}
              centerValue={formatMoney(periodTotal)}
              centerLabel={t("common.groupCount", {
                count: data?.groupCount ?? groups.filter((g) => g.groupId !== "").length,
              })}
              // The uncategorised wedge has no group to filter by, so it is not a link.
              onPressSegment={(segment) => {
                if (segment.id !== "") openGroup(segment.id);
              }}
            />
          </View>
        </Card>

        {templates.length > 0 ? (
          <View>
            <Kicker className="mb-n3 ml-n1">
              {t("home.quickTemplates", { name: templateOwner })}
            </Kicker>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={{ gap: nocturne.space.n2 }}
            >
              {templates.map((template) => (
                <Chip
                  key={template.id}
                  label={template.label}
                  icon={iconOr(template.icon)}
                  amount={fromWire(template.amount, currency)}
                  // Tap logs it straight away — the optimistic path in useLogTemplate; a
                  // long press opens the add screen prefilled instead.
                  onPress={() => logTemplate.mutate({ templateId: template.id })}
                  onLongPress={() => router.push(`/(app)/add?templateId=${template.id}`)}
                />
              ))}
            </ScrollView>
          </View>
        ) : null}

        {groups.length === 0 ? (
          <EmptyState
            icon="chart-donut"
            title={t("home.emptyTitle")}
            body={t("home.emptyBody")}
            action={{ label: t("home.addTransaction"), onPress: () => router.push("/(app)/add") }}
          />
        ) : (
          <View className="gap-n2">
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
                  // Spending with no category is a row so the shares add up, but there is no
                  // group filter that would show it — tapping would open the whole feed
                  // unfiltered, which reads as "the filter was ignored".
                  onPress={group.groupId === "" ? undefined : () => openGroup(group.groupId)}
                />
              );
            })}
          </View>
        )}
      </ScrollView>

      <Fab label={t("home.addTransaction")} onPress={() => router.push("/(app)/add")} />

      <Drawer
        visible={menuOpen}
        onClose={() => setMenuOpen(false)}
        account={{ name: family.data?.family?.name ?? "", email: family.data?.family?.name ?? "" }}
        household={{
          name: family.data?.family?.name ?? "",
          balance: fromWire(data?.headlineBalance, currency),
        }}
        scopes={[
          { id: "family", label: t("home.scopeFamily") },
          ...members.map((member) => ({ id: member.id, label: member.name })),
        ]}
        activeScopeId={scope.kind === "family" ? "family" : scope.id}
        onSelectScope={(id) => {
          setScope(id === "family" ? { kind: "family" } : { kind: "member", id });
          setMenuOpen(false);
        }}
        items={drawerItems}
        activeId="home"
        onSelect={(item) => {
          setMenuOpen(false);
          if (item.href && item.id !== "home") router.push(item.href);
        }}
      />
    </Screen>
  );
}

/** "Серпень 2026", "29 серпня, сб", "8/24 – 8/30" — whatever the current granularity names. */
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
