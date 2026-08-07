import { periodRange, useSummary, useTransactions, type DateRange } from "@fm/api";
import type { Transaction } from "@fm/sdk/finance/v1/finance_pb";
import { TransactionType } from "@fm/sdk/finance/v1/finance_pb";
import { Card, Dot, EmptyState, ErrorState, Loading } from "@fm/ui";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { FlatList, Pressable, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { Amount } from "../../components/Amount";
import { PeriodHeader, type SteppablePeriod } from "../../components/PeriodHeader";

/** The ledger: period totals on top, transactions below, one tap to add. */
export default function LedgerScreen() {
  const router = useRouter();
  const [kind, setKind] = useState<SteppablePeriod>("month");
  const [range, setRange] = useState<DateRange>(() => periodRange("month"));

  const summary = useSummary(range);
  const list = useTransactions({ range });

  const transactions = useMemo(
    () => list.data?.pages.flatMap((p) => p.transactions) ?? [],
    [list.data],
  );

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="gap-md p-lg">
        <PeriodHeader
          kind={kind}
          range={range}
          onChange={(k, r) => {
            setKind(k);
            setRange(r);
          }}
        />

        <Card className="flex-row justify-between p-lg">
          <Total label="Income" value={summary.data?.income} type={TransactionType.INCOME} />
          <Total label="Expenses" value={summary.data?.expense} type={TransactionType.EXPENSE} />
          <Total label="Balance" value={summary.data?.balance} />
        </Card>
      </View>

      {list.isPending ? (
        <Loading />
      ) : list.isError ? (
        <ErrorState message={list.error.message} onRetry={() => void list.refetch()} />
      ) : (
        <FlatList
          data={transactions}
          keyExtractor={(t) => t.id}
          contentContainerClassName="px-lg pb-2xl gap-xs"
          renderItem={({ item }) => (
            <TransactionRow
              transaction={item}
              onPress={() => router.push(`/(app)/transaction/${item.id}`)}
            />
          )}
          ListEmptyComponent={
            <EmptyState
              title="Nothing here yet"
              hint="Add your first transaction with the + button."
            />
          }
          onEndReachedThreshold={0.4}
          onEndReached={() => {
            if (list.hasNextPage && !list.isFetchingNextPage) void list.fetchNextPage();
          }}
          refreshing={list.isRefetching}
          onRefresh={() => void list.refetch()}
        />
      )}

      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Add transaction"
        onPress={() => router.push("/(app)/transaction/new")}
        className="absolute bottom-xl right-xl h-14 w-14 items-center justify-center rounded-full bg-primary"
      >
        <Text className="text-title text-primary-fg">+</Text>
      </Pressable>
    </SafeAreaView>
  );
}

function Total({
  label,
  value,
  type,
}: {
  label: string;
  value: Parameters<typeof Amount>[0]["value"];
  type?: TransactionType;
}) {
  return (
    <View className="gap-xs">
      <Text className="text-caption text-muted dark:text-muted-dark">{label}</Text>
      <Amount value={value} type={type} />
    </View>
  );
}

function TransactionRow({
  transaction,
  onPress,
}: {
  transaction: Transaction;
  onPress: () => void;
}) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress}>
      <Card className="flex-row items-center gap-md p-md">
        <Dot color={colorFor(transaction.type)} />
        <View className="flex-1">
          <Text className="text-body text-fg dark:text-fg-dark" numberOfLines={1}>
            {transaction.note !== "" ? transaction.note : labelFor(transaction.type)}
          </Text>
          <Text className="text-caption text-muted dark:text-muted-dark">
            {transaction.occurredOn}
          </Text>
        </View>
        <Amount value={transaction.amount} type={transaction.type} />
      </Card>
    </Pressable>
  );
}

function colorFor(type: TransactionType): string {
  switch (type) {
    case TransactionType.INCOME:
      return "#2F855A";
    case TransactionType.EXPENSE:
      return "#C53030";
    default:
      return "#2B6CB0";
  }
}

function labelFor(type: TransactionType): string {
  switch (type) {
    case TransactionType.INCOME:
      return "Income";
    case TransactionType.EXPENSE:
      return "Expense";
    default:
      return "Transfer";
  }
}
