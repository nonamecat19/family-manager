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
import {
  Button,
  dayHeading,
  Fab,
  Field,
  formatMoney,
  IconButton,
  Chip,
  Kicker,
  MoneyText,
  nocturne,
  PERIOD_TABS,
  PeriodTabs,
  Screen,
  ScreenHeader,
  SegmentedTabs,
  EmptyState,
  type PeriodTab,
} from "@/components/nocturne";
import { FilterSheet } from "@/components/screens/transactions/FilterSheet.tsx";
import { periodLabel } from "@/components/screens/transactions/periodLabel.ts";
import { TransactionRow } from "@/components/screens/transactions/TransactionRow.tsx";

type Kind = "expense" | "income";

/**
 * Screen 07 — the transaction feed.
 *
 * Day sections with their own subtotal, and the payer's avatar on every single row: the
 * household's question about a list of spending is "who?", so the answer is never a tap away.
 *
 * The feed is scoped to the family, which is what keeps private accounts out of it — the
 * service excludes them from a family scope, and this screen never re-adds them (the filter
 * sheet offers shared accounts only).
 */
export default function TransactionsScreen() {
  const { t } = useI18n();
  const router = useRouter();

  // Home taps a group row and screen 04 taps a category; both land here with the feed already
  // narrowed. The narrowing is shown as a chip that can be cleared, so the screen is never
  // stuck filtered by something the user cannot see. The tab and the window travel with the
  // tap too — a group read under "Доходи" in July opens the July income feed.
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
    // "custom" is a range, not an anchor, and a deep link carries only the anchor — so a link
    // from a custom window opens the month around it rather than an unrepresentable period.
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
    groupIds: focusGroupId === "" ? [] : [focusGroupId],
    categoryIds: focusCategoryId === "" ? [] : [focusCategoryId],
    query: query.trim(),
  });

  // One page boundary can split a day in two; the feed shows one section per date regardless.
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

  // The tint follows the category's position in the tree, so one category keeps one colour
  // across every day section (and matches the donut on screen 06).
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

  // The chip names whatever the feed was narrowed to; an id with no match (a stale deep link)
  // leaves it empty rather than showing a raw uuid.
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

  const steppable = period.granularity !== "custom" ? (period as SteppablePeriod) : null;
  const atToday = range.to >= toISODate(new Date());

  const changeTab = (next: PeriodTab) => {
    setTab(next);
    // Switching granularity keeps the day the user is looking at; "Період" freezes the window
    // it was showing, since there is no range picker to open yet.
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
      <ScreenHeader
        gradient
        className="overflow-hidden rounded-b-[24px]"
        title={t("transactions.title")}
        subtitle={t("common.family")}
        leading={{ icon: "arrow-left", label: t("common.back"), onPress: () => router.back() }}
        trailing={[
          {
            icon: "magnifying-glass",
            label: t("transactions.search"),
            onPress: () => setSearchOpen((open) => !open),
            badge: query.trim() !== "",
          },
          {
            icon: "funnel",
            label: t("transactions.filter"),
            onPress: () => setFilterOpen(true),
            badge: memberIds.length + accountIds.length > 0,
          },
        ]}
      >
        <SegmentedTabs<Kind>
          className="mt-n4"
          value={kind}
          onChange={setKind}
          options={[
            { value: "expense", label: t("common.expenses") },
            { value: "income", label: t("common.income") },
          ]}
        />
      </ScreenHeader>

      {searchOpen ? (
        <View className="px-n5 pt-n3">
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
        <View className="flex-row px-n5 pt-n3">
          <Chip
            label={focusLabel}
            icon="x"
            variant="outline"
            onPress={() => setCleared(true)}
          />
        </View>
      )}

      <View className="flex-1 px-n4 pt-n4">
        <PeriodTabs value={tab} onChange={changeTab} />

        <View className="mt-n3 flex-row items-center justify-between">
          <View className="flex-row items-center">
            {steppable ? (
              <IconButton
                icon="caret-left"
                label={t("common.back")}
                onPress={() => step(-1)}
                size={16}
              />
            ) : null}
            <Text className="text-[13px] text-fg">{periodLabel(t, period, range)}</Text>
            {steppable ? (
              <IconButton
                icon="caret-right"
                label={t("common.next")}
                // Stepping past the current period would only ever show an empty feed.
                onPress={() => {
                  if (!atToday) step(1);
                }}
                size={16}
                color={atToday ? nocturne.neutral[800] : nocturne.text}
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
            <View className="mb-n4">
              <Kicker className="mb-n3 ml-n2">
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
        onToggleMember={(id) => setMemberIds((current) => toggle(current, id))}
        onToggleAccount={(id) => setAccountIds((current) => toggle(current, id))}
        onClear={() => {
          setMemberIds([]);
          setAccountIds([]);
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

/** The list and the three states it can be in instead. */
function Feed({ days, pending, empty, error, onRetry, fetchingMore, onEndReached, renderDay }: FeedProps) {
  const { t } = useI18n();

  if (pending) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
      </View>
    );
  }

  if (error) {
    const shown = toDisplayError(error, t("common.loadFailed"));
    return (
      <View className="flex-1 justify-center gap-n3 px-n4">
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

  if (empty) {
    return (
      <View className="flex-1 justify-center">
        <EmptyState
          icon="receipt"
          title={t("transactions.emptyTitle")}
          body={t("transactions.emptyBody")}
        />
      </View>
    );
  }

  return (
    <FlatList
      data={days}
      className="mt-n4"
      keyExtractor={(day) => day.date}
      renderItem={({ item, index }) => renderDay(item, index === days.length - 1)}
      showsVerticalScrollIndicator={false}
      onEndReached={onEndReached}
      onEndReachedThreshold={0.4}
      ListFooterComponent={
        fetchingMore ? (
          <Text className="pb-n5 text-center text-[13px] text-neutral-500">
            {t("common.loadingEllipsis")}
          </Text>
        ) : null
      }
    />
  );
}

/** The tab a deep link asked for. An unknown or absent value is a month, which is the design's
 * own default and the only granularity every screen agrees on. */
function initialTab(granularity: string | undefined): PeriodTab {
  return PERIOD_TABS.includes(granularity as PeriodTab) ? (granularity as PeriodTab) : "month";
}
