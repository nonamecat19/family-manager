import { periodRange, toDisplayError, type DateRange, useCategoryBreakdown } from "@fm/api";
import { TransactionType } from "@fm/sdk/finance/v1/finance_pb";
import { categoryColor, categoryPalette } from "@fm/theme";
import { Card, Dot, EmptyState, ErrorState, Loading, PieChart } from "@fm/ui";
import { useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { Amount } from "../../components/Amount";
import { PeriodHeader, type SteppablePeriod } from "../../components/PeriodHeader";

/** Where the money went: one donut, one row per category, for the chosen period. */
export default function ReportsScreen() {
  const [kind, setKind] = useState<SteppablePeriod>("month");
  const [range, setRange] = useState<DateRange>(() => periodRange("month"));
  const [type, setType] = useState<TransactionType>(TransactionType.EXPENSE);

  const breakdown = useCategoryBreakdown(range, type);
  const slices = breakdown.data?.slices ?? [];

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <ScrollView contentContainerClassName="gap-lg p-lg pb-2xl">
        <PeriodHeader
          kind={kind}
          range={range}
          onChange={(k, r) => {
            setKind(k);
            setRange(r);
          }}
        />

        <View className="flex-row gap-xs">
          <Toggle
            label="Expenses"
            active={type === TransactionType.EXPENSE}
            onPress={() => setType(TransactionType.EXPENSE)}
          />
          <Toggle
            label="Income"
            active={type === TransactionType.INCOME}
            onPress={() => setType(TransactionType.INCOME)}
          />
        </View>

        {breakdown.isPending ? (
          <Loading />
        ) : breakdown.isError ? (
          <ErrorState
            {...toDisplayError(breakdown.error, "Could not load this report.")}
            onRetry={() => void breakdown.refetch()}
          />
        ) : slices.length === 0 ? (
          <EmptyState
            title="Nothing in this period"
            hint="Pick another period, or add a transaction."
          />
        ) : (
          <>
            <View className="items-center">
              <PieChart
                slices={slices.map((s, i) => ({
                  key: s.categoryId,
                  share: s.share,
                  color: categoryColor(s.color, i, categoryPalette),
                }))}
              >
                <View className="items-center">
                  <Text className="text-caption text-muted dark:text-muted-dark">Total</Text>
                  <Amount value={breakdown.data?.total} />
                </View>
              </PieChart>
            </View>

            <View className="gap-xs">
              {slices.map((s, i) => (
                <Card key={s.categoryId} className="flex-row items-center gap-md p-md">
                  <Dot color={categoryColor(s.color, i, categoryPalette)} />
                  <View className="flex-1">
                    <Text className="text-body text-fg dark:text-fg-dark">{s.categoryName}</Text>
                    <Text className="text-caption text-muted dark:text-muted-dark">
                      {Math.round(s.share * 100)}% · {s.transactionCount}{" "}
                      {s.transactionCount === 1 ? "entry" : "entries"}
                    </Text>
                  </View>
                  <Amount value={s.total} type={type} />
                </Card>
              ))}
            </View>
          </>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

function Toggle({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected: active }}
      onPress={onPress}
      className={`flex-1 rounded-md py-sm ${
        active ? "bg-primary" : "border border-border dark:border-border-dark"
      }`}
    >
      <Text
        className={`text-center text-body ${active ? "text-primary-fg" : "text-muted dark:text-muted-dark"}`}
      >
        {label}
      </Text>
    </Pressable>
  );
}
