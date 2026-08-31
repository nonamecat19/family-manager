import {
  BudgetTargetFilter,
  fromWire,
  type SeriesBucket,
  SeriesStacking,
  toDisplayError,
  TransactionKind,
  useBudgets,
  useCategoryTree,
  useFamily,
  useSpendingSeries,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  type BarPoint,
  type BarSeries,
  BudgetBar,
  Button,
  Card,
  ChartLegend,
  Drawer,
  EmptyState,
  memberColor,
  monthNameLower,
  nocturne,
  Screen,
  ScreenHeader,
  SegmentedTabs,
  StackedBarSeries,
  useDrawerItems,
} from "@/components/nocturne";

/**
 * Screen 09 — Графіки.
 *
 * Two cards: seven buckets of spending stacked by member (the granularity strip picks the
 * bucket width), and the household's group budgets for the current month. Both read
 * family-scoped aggregates, so private accounts are already out of every total the server
 * returns — this screen never sums anything itself.
 */

const GRANULARITIES = ["year", "month", "week", "day"] as const;
type Granularity = (typeof GRANULARITIES)[number];

/** The design draws seven columns; the current bucket is the seventh. */
const BUCKET_COUNT = 7;

type KindTab = "total" | "expenses" | "income";

const KIND_WIRE: Record<KindTab, TransactionKind> = {
  total: TransactionKind.UNSPECIFIED,
  expenses: TransactionKind.EXPENSE,
  income: TransactionKind.INCOME,
};

export default function ChartsScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [kind, setKind] = useState<KindTab>("expenses");
  const [granularity, setGranularity] = useState<Granularity>("month");
  const [menuOpen, setMenuOpen] = useState(false);
  const drawerItems = useDrawerItems();

  const series = useSpendingSeries({
    granularity,
    bucketCount: BUCKET_COUNT,
    kind: KIND_WIRE[kind],
    stackedBy: SeriesStacking.MEMBER,
  });
  const budgets = useBudgets({ target: BudgetTargetFilter.GROUP });
  const tree = useCategoryTree();
  const family = useFamily();

  const chart = useChartData(series.data?.buckets ?? [], t("common.family"));
  const groupNames = useMemo(() => {
    const byId = new Map<string, string>();
    for (const node of tree.data ?? []) {
      if (node.group) byId.set(node.group.id, node.group.name);
    }
    return byId;
  }, [tree.data]);

  const budgetRows = useMemo(
    () =>
      (budgets.data?.budgets ?? []).map((status) => ({
        id: status.budget?.id ?? "",
        label: groupNames.get(status.budget?.groupId ?? "") ?? "",
        spentMinor: fromWire(status.spent).amountMinor,
        limitMinor: fromWire(status.budget?.limit).amountMinor,
      })),
    [budgets.data, groupNames],
  );

  const month = new Date().getMonth() + 1;
  const pending = series.isPending || budgets.isPending;
  const failed = series.isError || budgets.isError;
  const hasChart = chart.points.some((p) => p.values.some((v) => v > 0));
  const empty = !pending && !failed && !hasChart && budgetRows.length === 0;

  const retry = () => {
    void series.refetch();
    void budgets.refetch();
    void tree.refetch();
  };

  return (
    <Screen>
      <ScreenHeader
        gradient
        title={t("charts.title")}
        leading={{ icon: "list", label: t("nav.menu"), onPress: () => setMenuOpen(true) }}
        trailing={{
          icon: "users-three",
          label: t("nav.household"),
          onPress: () => router.push("/(app)/household"),
        }}
      >
        <SegmentedTabs<KindTab>
          className="mt-n4"
          value={kind}
          onChange={setKind}
          options={[
            { value: "total", label: t("common.total") },
            { value: "expenses", label: t("common.expenses") },
            { value: "income", label: t("common.income") },
          ]}
        />
      </ScreenHeader>

      {pending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[13px] text-neutral-500">{t("gate.preparing")}</Text>
        </View>
      ) : failed ? (
        <ErrorState error={series.error ?? budgets.error} onRetry={retry} />
      ) : empty ? (
        <View className="flex-1 justify-center">
          <EmptyState icon="chart-bar" title={t("charts.emptyTitle")} body={t("charts.emptyBody")} />
        </View>
      ) : (
        <ScrollView
          className="flex-1"
          contentContainerClassName="gap-n3 px-n4 pb-n6 pt-n4"
          showsVerticalScrollIndicator={false}
        >
          <Card>
            <GranularityStrip value={granularity} onChange={setGranularity} />
            <View accessibilityLabel={t("charts.byMember")} className="mt-n5">
              <StackedBarSeries
                points={chart.points}
                series={chart.series}
                height={170}
                legend={false}
                activeIndex={series.data?.currentBucketIndex}
              />
            </View>
            <ChartLegend series={chart.series} className="mt-n4 justify-center" />
          </Card>

          {budgetRows.length > 0 ? (
            <Card>
              <View className="mb-n4 flex-row items-baseline justify-between">
                <Text className="text-[13px] font-medium text-fg">
                  {t("charts.groupBudgets", { month: monthNameLower(t, month) })}
                </Text>
                <Text className="text-[11px] text-neutral-600">
                  {t("common.xOfY", {
                    done: budgets.data?.withinLimitCount ?? 0,
                    total: budgets.data?.totalCount ?? 0,
                  })}
                </Text>
              </View>
              <View className="gap-n3">
                {budgetRows.map((row) => (
                  <BudgetBar
                    key={row.id}
                    label={row.label}
                    spentMinor={row.spentMinor}
                    limitMinor={row.limitMinor}
                    valueLabel={row.limitMinor > 0 ? undefined : t("common.noBudget")}
                  />
                ))}
              </View>
            </Card>
          ) : null}
        </ScrollView>
      )}

      <Drawer
        visible={menuOpen}
        onClose={() => setMenuOpen(false)}
        // No RPC tells a screen who the signed-in person is (see the report): the panel names
        // the household rather than inventing a person.
        account={{ name: family.data?.family?.name ?? "", email: "" }}
        household={{ name: family.data?.family?.name ?? "" }}
        items={drawerItems}
        activeId="charts"
        onSelect={(item) => {
          setMenuOpen(false);
          if (item.href && item.id !== "charts") router.push(item.href);
        }}
      />
    </Screen>
  );
}

/** рік · місяць · тиждень · день — the bucket width, not the period being viewed, so this is
 * the design's own lowercase strip rather than the kit's PeriodTabs. */
function GranularityStrip({
  value,
  onChange,
}: {
  value: Granularity;
  onChange: (next: Granularity) => void;
}) {
  const { t } = useI18n();
  const labels: Record<Granularity, string> = {
    year: t("charts.granularity.year"),
    month: t("charts.granularity.month"),
    week: t("charts.granularity.week"),
    day: t("charts.granularity.day"),
  };
  return (
    <View className="flex-row justify-center gap-n5">
      {GRANULARITIES.map((option) => {
        const active = option === value;
        return (
          <Pressable
            key={option}
            accessibilityRole="tab"
            accessibilityLabel={labels[option]}
            accessibilityState={{ selected: active }}
            onPress={() => onChange(option)}
            hitSlop={6}
            className="items-center"
          >
            <Text
              className={`pb-[3px] text-[12px] ${active ? "font-medium text-accent-400" : "text-neutral-500"}`}
            >
              {labels[option]}
            </Text>
            <View
              className="h-[2px] w-full rounded-full"
              style={{ backgroundColor: active ? nocturne.accent.DEFAULT : "transparent" }}
            />
          </Pressable>
        );
      })}
    </View>
  );
}

/**
 * Buckets → the chart's two arrays. Members are ordered by first appearance across the whole
 * window, so a member who spent nothing in the newest bucket keeps their colour and their
 * place in the legend. A household the server did not split (one member, or a kind nobody
 * logged) falls back to a single "Родина" series drawn from the bucket totals.
 */
function useChartData(
  buckets: readonly SeriesBucket[],
  familyLabel: string,
): { points: BarPoint[]; series: BarSeries[] } {
  return useMemo(() => {
    const order: string[] = [];
    const meta = new Map<string, { label: string; colorStep: number }>();
    for (const bucket of buckets) {
      for (const segment of bucket.segments) {
        if (meta.has(segment.key)) continue;
        meta.set(segment.key, { label: segment.label, colorStep: segment.colorStep });
        order.push(segment.key);
      }
    }

    if (order.length === 0) {
      return {
        series: [{ id: "family", label: familyLabel, color: memberColor(0) }],
        points: buckets.map((bucket) => ({
          label: bucket.label,
          values: [fromWire(bucket.total).amountMinor],
        })),
      };
    }

    const series: BarSeries[] = order.map((key, index) => {
      const entry = meta.get(key);
      return {
        id: key,
        label: entry?.label ?? "",
        color: memberColor(entry?.colorStep ?? index),
      };
    });
    const points: BarPoint[] = buckets.map((bucket) => ({
      label: bucket.label,
      values: order.map((key) => {
        const segment = bucket.segments.find((s) => s.key === key);
        return segment ? fromWire(segment.amount).amountMinor : 0;
      }),
    }));
    return { series, points };
  }, [buckets, familyLabel]);
}

function ErrorState({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const { t } = useI18n();
  const shown = toDisplayError(error, t("common.loadFailed"));
  return (
    <View className="flex-1 justify-center gap-n4 px-n6">
      <Text className="text-[13.5px] leading-[21px] text-neutral-500">{shown.message}</Text>
      {shown.reference ? (
        <Text className="text-[12px] text-neutral-600">
          {t("common.errorReference", { ref: shown.reference })}
        </Text>
      ) : null}
      <Button title={t("common.tryAgain")} onPress={onRetry} />
    </View>
  );
}
