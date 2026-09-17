import {
  currentPeriod,
  familyScope,
  fromWire,
  periodWindow,
  stepPeriod,
  toDisplayError,
  toISODate,
  TransactionKind,
  TransactionType,
  useAccounts,
  useCategoryTree,
  useDeleteTransaction,
  useFinanceMembers,
  useFinanceSettings,
  useTransactionFeed,
  type DaySection,
  type PeriodInput,
  type SteppablePeriod,
} from "@fm/api";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Alert, FlatList, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { AmountChip, dayHeading, Fab, formatMoney, MoneyText, PERIOD_TABS, PeriodTabs, SegmentedTabs, type PeriodTab } from "@/components/kit";
import { FilterSheet } from "@/components/screens/transactions/FilterSheet.tsx";
import { periodLabel } from "@/components/screens/transactions/periodLabel.ts";
import { TransactionRow } from "@/components/screens/transactions/TransactionRow.tsx";
import { Button, Field, IconButton, Kicker, organic, Screen, ScreenHeader, EmptyState } from "@fm/ui";

type Kind = "expense" | "income";

export default function TransactionsScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const params = useLocalSearchParams<{
    groupId?: string;
    categoryId?: string;
    kind?: string;
    granularity?: string;
    anchor?: string;
  }>();
  const [cleared, setCleared] = useState(false);
  const focusGroupId = cleared || typeof params.groupId !== "string" ? "" : params.groupId;
  const focusCategoryId = cleared || typeof params.categoryId !== "string" ? "" : params.categoryId;

  const [kind, setKind] = useState<Kind>(params.kind === "income" ? "income" : "expense");
  const [tab, setTab] = useState<PeriodTab>(initialTab(params.granularity));
  const [period, setPeriod] = useState<PeriodInput>(() => {
    const granularity = initialTab(params.granularity);
    if (typeof params.anchor !== "string" || params.anchor === "" || granularity === "custom") {
      return currentPeriod(granularity === "custom" ? "month" : granularity);
    }
    return { granularity, anchor: params.anchor };
  });
  const [searchOpen, setSearchOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [filterOpen, setFilterOpen] = useState(false);
  const [memberIds, setMemberIds] = useState<readonly string[]>([]);
  const [accountIds, setAccountIds] = useState<readonly string[]>([]);
  const [groupIds, setGroupIds] = useState<readonly string[]>([]);

  const settings = useFinanceSettings();
  const members = useFinanceMembers();
  const accounts = useAccounts();
  const tree = useCategoryTree();
  const remove = useDeleteTransaction();

  const currency = settings.data?.baseCurrencyCode || "UAH";
  const range = useMemo(() => periodWindow(period), [period]);

  const feed = useTransactionFeed({
    scope: familyScope,
    period,
    kind: kind === "income" ? TransactionKind.INCOME : TransactionKind.EXPENSE,
    memberIds,
    accountIds,
    groupIds: focusGroupId === "" || groupIds.includes(focusGroupId) ? groupIds : [...groupIds, focusGroupId],
    categoryIds: focusCategoryId === "" ? [] : [focusCategoryId],
    query: query.trim(),
  });

  const days = useMemo(() => {
    const sections: DaySection[] = [];
    const byDate = new Map<string, DaySection>();
    for (const page of feed.data?.pages ?? []) {
      for (const day of page.days) {
        const seen = byDate.get(day.date);
        if (seen) {
          seen.transactions = [...seen.transactions, ...day.transactions];
          continue;
        }
        const copy: DaySection = { ...day, transactions: [...day.transactions] };
        byDate.set(day.date, copy);
        sections.push(copy);
      }
    }
    return sections;
  }, [feed.data]);

  const periodTotal = feed.data?.pages[0]?.periodTotal;

  const memberIndex = useMemo(() => {
    const map = new Map<string, { name: string; index: number }>();
    (members.data ?? []).forEach((member, index) => {
      map.set(member.userId, { name: member.displayName, index });
    });
    return map;
  }, [members.data]);

  const accountNames = useMemo(() => {
    const map = new Map<string, string>();
    for (const account of [...(accounts.data?.shared ?? []), ...(accounts.data?.privateOwn ?? [])]) {
      map.set(account.id, account.name);
    }
    return map;
  }, [accounts.data]);

  const categories = useMemo(() => {
    const map = new Map<string, { name: string; icon: string; group: string; index: number }>();
    let index = 0;
    for (const node of tree.data ?? []) {
      for (const category of node.categories) {
        map.set(category.id, {
          name: category.name,
          icon: category.icon,
          group: node.group?.name ?? "",
          index: index++,
        });
      }
    }
    return map;
  }, [tree.data]);

  const focused = focusCategoryId !== "" || focusGroupId !== "";
  const focusLabel =
    (focusCategoryId === "" ? "" : (categories.get(focusCategoryId)?.name ?? "")) ||
    (focusGroupId === ""
      ? ""
      : ((tree.data ?? []).find((node) => node.group?.id === focusGroupId)?.group?.name ?? "")) ||
    (focused ? t("transactions.focusUnknown") : "");

  const sharedAccountOptions = useMemo(
    () => (accounts.data?.shared ?? []).map((account) => ({ id: account.id, label: account.name })),
    [accounts.data],
  );
  const memberOptions = useMemo(
    () => (members.data ?? []).map((member) => ({ id: member.userId, label: member.displayName })),
    [members.data],
  );
  const groupKind = kind === "income" ? TransactionKind.INCOME : TransactionKind.EXPENSE;
  const groupOptions = useMemo(
    () =>
      (tree.data ?? [])
        .filter((node) => node.group?.kind === groupKind)
        .map((node) => ({ id: node.group?.id ?? "", label: node.group?.name ?? "" })),
    [tree.data, groupKind],
  );

  const steppable = period.granularity !== "custom" ? (period as SteppablePeriod) : null;
  const atToday = range.to >= toISODate(new Date());

  const changeTab = (next: PeriodTab) => {
    setTab(next);
    setPeriod(
      next === "custom"
        ? { granularity: "custom", range }
        : { granularity: next, anchor: range.from },
    );
  };

  const step = (by: number) => {
    if (steppable) setPeriod(stepPeriod(steppable, by));
  };

  const toggle = (values: readonly string[], id: string) =>
    values.includes(id) ? values.filter((value) => value !== id) : [...values, id];

  const confirmDelete = (transactionId: string, title: string) => {
    Alert.alert(t("transactions.deleteConfirm"), title, [
      { text: t("common.cancel"), style: "cancel" },
      {
        text: t("common.delete"),
        style: "destructive",
        onPress: () => remove.mutate(transactionId),
      },
    ]);
  };

  return (
    <Screen edges={["top"]}>
      <View className="gap-lg px-5.5 pt-sm">
        <ScreenHeader
          title={t("transactions.title")}
          kicker={t("common.family")}
          onBack={focused ? () => router.back() : undefined}
          backLabel={t("common.back")}
          actions={
            <>
              <IconButton
                icon="magnifying-glass"
                label={t("transactions.search")}
                onPress={() => setSearchOpen((open) => !open)}
                badge={query.trim() !== ""}
              />
              <IconButton
                icon="funnel"
                label={t("transactions.filter")}
                onPress={() => setFilterOpen(true)}
                badge={memberIds.length + accountIds.length + groupIds.length > 0}
              />
            </>
          }
        />
        <SegmentedTabs<Kind>
          value={kind}
          onChange={setKind}
          options={[
            { value: "expense", label: t("common.expenses") },
            { value: "income", label: t("common.income") },
          ]}
        />
      </View>

      {searchOpen ? (
        <View className="px-5.5 pt-sm">
          <Field
            label={t("transactions.search")}
            value={query}
            onChangeText={setQuery}
            autoFocus
            returnKeyType="search"
          />
        </View>
      ) : null}

      {!focused ? null : (
        <View className="flex-row px-5.5 pt-sm">
          <AmountChip
            label={focusLabel}
            icon="x"
            variant="outline"
            onPress={() => setCleared(true)}
          />
        </View>
      )}

      <View className="flex-1 px-2.8 pt-2.8">
        <PeriodTabs value={tab} onChange={changeTab} />

        <View className="mt-2.1 flex-row items-center justify-between">
          <View className="flex-row items-center">
            {steppable ? (
              <IconButton
                icon="caret-left"
                label={t("common.back")}
                onPress={() => step(-1)}
                size={16}
              />
            ) : null}
            <Text className="text-13 text-fg">{periodLabel(t, period, range)}</Text>
            {steppable ? (
              <IconButton
                icon="caret-right"
                label={t("common.next")}
                onPress={() => {
                  if (!atToday) step(1);
                }}
                size={16}
                color={atToday ? organic.neutral[800] : organic.text}
              />
            ) : null}
          </View>
          {periodTotal ? (
            <MoneyText value={fromWire(periodTotal, currency)} size={13} weight="medium" />
          ) : null}
        </View>

        <Feed
          days={days}
          empty={feed.isSuccess && days.length === 0}
          pending={feed.isPending}
          error={feed.isError ? feed.error : null}
          onRetry={() => void feed.refetch()}
          fetchingMore={feed.isFetchingNextPage}
          onEndReached={() => {
            if (feed.hasNextPage && !feed.isFetchingNextPage) void feed.fetchNextPage();
          }}
          renderDay={(day, last) => (
            <View className="mb-2.8">
              <Kicker className="mb-2.1 ml-1.4">
                {t("transactions.dayHeading", {
                  date: day.weekdayLabel || dayHeading(t, day.date),
                  amount: formatMoney(fromWire(day.dayTotal, currency)),
                })}
              </Kicker>
              <View className="overflow-hidden rounded-lg bg-surface">
                {day.transactions.map((transaction, index) => {
                  const category = categories.get(transaction.categoryId);
                  const member = memberIndex.get(transaction.memberId);
                  const title = category?.name ?? transaction.note ?? "";
                  return (
                    <TransactionRow
                      key={transaction.id}
                      title={title}
                      meta={t("transactions.rowMeta", {
                        account: accountNames.get(transaction.accountId) ?? "",
                        category: transaction.merchant || category?.group || category?.name || "",
                      })}
                      icon={category?.icon}
                      iconIndex={category?.index ?? 0}
                      memberName={member?.name ?? ""}
                      memberIndex={member?.index ?? 0}
                      amount={fromWire(transaction.amount, currency)}
                      income={transaction.type === TransactionType.INCOME}
                      templateLabel={
                        transaction.templateId ? t("transactions.templateBadge") : undefined
                      }
                      onPress={() =>
                        router.push({
                          pathname: "/(app)/add",
                          params: { transactionId: transaction.id },
                        })
                      }
                      onLongPress={() => confirmDelete(transaction.id, title)}
                      divider={index < day.transactions.length - 1}
                    />
                  );
                })}
              </View>
              {last ? <View className="h-[84px]" /> : null}
            </View>
          )}
        />
      </View>

      <FilterSheet
        visible={filterOpen}
        onClose={() => setFilterOpen(false)}
        members={memberOptions}
        accounts={sharedAccountOptions}
        memberIds={memberIds}
        accountIds={accountIds}
        groups={groupOptions}
        groupIds={groupIds}
        onToggleMember={(id) => setMemberIds((current) => toggle(current, id))}
        onToggleAccount={(id) => setAccountIds((current) => toggle(current, id))}
        onToggleGroup={(id) => setGroupIds((current) => toggle(current, id))}
        onClear={() => {
          setMemberIds([]);
          setAccountIds([]);
          setGroupIds([]);
        }}
      />

      <Fab label={t("add.title")} onPress={() => router.push("/(app)/add")} />
    </Screen>
  );
}

interface FeedProps {
  days: readonly DaySection[];
  pending: boolean;
  empty: boolean;
  error: unknown;
  onRetry: () => void;
  fetchingMore: boolean;
  onEndReached: () => void;
  renderDay: (day: DaySection, last: boolean) => React.ReactElement;
}

function Feed({ days, pending, empty, error, onRetry, fetchingMore, onEndReached, renderDay }: FeedProps) {
  const { t } = useI18n();

  if (pending) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-13 text-neutral-600">{t("common.loadingEllipsis")}</Text>
      </View>
    );
  }

  if (error) {
    const shown = toDisplayError(error, t("common.loadFailed"));
    return (
      <View className="flex-1 justify-center gap-2.1 px-2.8">
        <Text className="text-13.5 leading-[21px] text-neutral-600">{shown.message}</Text>
        {shown.reference ? (
          <Text className="text-12 text-neutral-600">
            {t("common.errorReference", { ref: shown.reference })}
          </Text>
        ) : null}
        <Button title={t("common.tryAgain")} onPress={onRetry} />
      </View>
    );
  }

  if (empty) {
    return (
      <View className="flex-1 justify-center">
        <EmptyState
          title={t("transactions.emptyTitle")}
          body={t("transactions.emptyBody")}
        />
      </View>
    );
  }

  return (
    <FlatList
      data={days}
      className="mt-2.8"
      keyExtractor={(day) => day.date}
      renderItem={({ item, index }) => renderDay(item, index === days.length - 1)}
      showsVerticalScrollIndicator={false}
      onEndReached={onEndReached}
      onEndReachedThreshold={0.4}
      ListFooterComponent={
        fetchingMore ? (
          <Text className="pb-4.2 text-center text-13 text-neutral-600">
            {t("common.loadingEllipsis")}
          </Text>
        ) : null
      }
    />
  );
}

function initialTab(granularity: string | undefined): PeriodTab {
  return PERIOD_TABS.includes(granularity as PeriodTab) ? (granularity as PeriodTab) : "month";
}
