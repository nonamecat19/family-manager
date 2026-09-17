import { LoadError } from "@/components/kit/LoadError.tsx";
import {
  BudgetTargetFilter,
  fromWire,
  type SeriesBucket,
  SeriesStacking,
  TransactionKind,
  useBudgets,
  useCategoryTree,
  useSpendingSeries,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { type BarPoint, type BarSeries, BudgetBar, Card, ChartLegend, memberColor, monthNameLower, SegmentedTabs, StackedBarSeries } from "@/components/kit";
import { EmptyState, IconButton, organic, Screen, ScreenHeader } from "@fm/ui";

const GRANULARITIES = ["year", "month", "week", "day"] as const;
type Granularity = (typeof GRANULARITIES)[number];

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

  const series = useSpendingSeries({
    granularity,
    bucketCount: BUCKET_COUNT,
    kind: KIND_WIRE[kind],
    stackedBy: SeriesStacking.MEMBER,
  });
  const budgets = useBudgets({ target: BudgetTargetFilter.GROUP });
  const tree = useCategoryTree();

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
      <View className="gap-lg px-5.5 pt-sm">
        <ScreenHeader
          title={t("charts.title")}
          onBack={() => router.back()}
          backLabel={t("common.back")}
          actions={
            <IconButton
              icon="users-three"
              label={t("nav.household")}
              onPress={() => router.push("/(app)/household")}
            />
          }
        />
        <SegmentedTabs<KindTab>
          value={kind}
          onChange={setKind}
          options={[
            { value: "total", label: t("common.total") },
            { value: "expenses", label: t("common.expenses") },
            { value: "income", label: t("common.income") },
          ]}
        />
      </View>

      {pending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-13 text-neutral-600">{t("gate.preparing")}</Text>
        </View>
      ) : failed ? (
        <LoadError error={series.error ?? budgets.error} onRetry={retry} />
      ) : empty ? (
        <View className="flex-1 justify-center">
          <EmptyState title={t("charts.emptyTitle")} body={t("charts.emptyBody")} />
        </View>
      ) : (
        <ScrollView
          className="flex-1"
          contentContainerClassName="gap-2.1 px-2.8 pb-5.6 pt-2.8"
          showsVerticalScrollIndicator={false}
        >
          <Card>
            <GranularityStrip value={granularity} onChange={setGranularity} />
            <View accessibilityLabel={t("charts.byMember")} className="mt-4.2">
              <StackedBarSeries
                points={chart.points}
                series={chart.series}
                height={170}
                legend={false}
                activeIndex={series.data?.currentBucketIndex}
              />
            </View>
            <ChartLegend series={chart.series} className="mt-2.8 justify-center" />
          </Card>

          {budgetRows.length > 0 ? (
            <Card>
              <View className="mb-2.8 flex-row items-baseline justify-between">
                <Text className="text-13 font-fig-med text-fg">
                  {t("charts.groupBudgets", { month: monthNameLower(t, month) })}
                </Text>
                <Text className="text-11 text-neutral-600">
                  {t("common.xOfY", {
                    done: budgets.data?.withinLimitCount ?? 0,
                    total: budgets.data?.totalCount ?? 0,
                  })}
                </Text>
              </View>
              <View className="gap-2.1">
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
    </Screen>
  );
}

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
    <View className="flex-row justify-center gap-4.2">
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
              className={`pb-0.75 text-12 ${active ? "font-fig-med text-accent-700" : "text-neutral-600"}`}
            >
              {labels[option]}
            </Text>
            <View
              className="h-[2px] w-full rounded-full"
              style={{ backgroundColor: active ? organic.accent.DEFAULT : "transparent" }}
            />
          </Pressable>
        );
      })}
    </View>
  );
}

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
