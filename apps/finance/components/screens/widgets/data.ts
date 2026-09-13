import {
  BudgetTargetFilter,
  TransactionKind,
  WidgetType,
  currentPeriod,
  periodWindow,
  useAccounts,
  useBudgets,
  useCategoryTree,
  useFinanceMembers,
  useGroupBreakdown,
  useHomeSummary,
  useMemberBreakdown,
  useTemplates,
  useTransactionFeed,
  useWidgetData,
  useWidgets,
  fromWire,
  type Account,
  type BudgetStatus,
  type MemberSpending,
  type Money,
  type WidgetPayload,
} from "@fm/api";
import { useMemo } from "react";

import { parseISO } from "@/components/nocturne";

export interface BudgetLine {
  id: string;
  label: string;
  spent: Money;
  limit: Money;
}

export interface RecentLine {
  id: string;
  label: string;
  amount: Money;
}

export interface TemplateCellModel {
  id: string;
  label: string;
  icon: string;
  amount: Money;
}

export interface QuickAddModel {
  ownerName: string;
  templates: TemplateCellModel[];
}

export interface MonthModel {
  label: string;
  total: Money;
  slices: { id: string; value: number }[];
  overspentCount: number;
}

export interface CategoryModel {
  name: string;
  icon: string;
  amount: Money;
  index: number;
}

export interface BudgetsAndFamilyModel {
  budgets: BudgetLine[];
  members: { id: string; name: string; spent: Money }[];
}

export interface AccountsModel {
  accounts: { id: string; name: string; balance: Money }[];
}

export interface WidgetPreviews {
  currency: string;
  quickAdd: QuickAddModel;
  month: MonthModel;
  category: CategoryModel | null;
  budgetsAndFamily: BudgetsAndFamilyModel;
  recent: RecentLine[];
  accounts: AccountsModel;
}

export interface WidgetPreviewsResult {
  previews: WidgetPreviews | null;
  isPending: boolean;
  error: unknown;
  refetch: () => void;
}

export type MonthLabel = (month: number) => string;

function payloadOf(payloads: readonly WidgetPayload[] | undefined, type: WidgetType) {
  return payloads?.find((p) => p.type === type)?.data;
}

export function useWidgetPreviews(monthLabel: MonthLabel): WidgetPreviewsResult {
  const period = useMemo(() => currentPeriod("month"), []);
  const kind = TransactionKind.EXPENSE;

  const home = useHomeSummary({ period, kind });
  const budgets = useBudgets({ target: BudgetTargetFilter.GROUP });
  const memberBreakdown = useMemberBreakdown(period, kind);
  const accounts = useAccounts();
  const templates = useTemplates();
  const members = useFinanceMembers();
  const tree = useCategoryTree();
  const feed = useTransactionFeed({ period, kind, pageSize: 10 });

  const topGroupId = home.data?.slices[0]?.groupId ?? "";
  const groupBreakdown = useGroupBreakdown(topGroupId, { period, kind });

  const placed = useWidgets();
  const placedIds = useMemo(() => (placed.data ?? []).map((w) => w.id), [placed.data]);
  const payloads = useWidgetData(placedIds);

  const queries = [home, budgets, memberBreakdown, accounts, templates, members, tree, feed];
  const isPending = queries.some((q) => q.isPending);
  const error = queries.find((q) => q.isError)?.error ?? null;

  const previews = useMemo<WidgetPreviews | null>(() => {
    if (!home.data || !budgets.data || !memberBreakdown.data || !accounts.data) return null;
    if (!templates.data || !members.data || !tree.data || !feed.data) return null;

    const currency = home.data.periodTotal?.currencyCode ?? "UAH";
    const data = payloads.data;

    const memberName = (id: string) =>
      members.data.find((m) => m.userId === id)?.displayName ?? "";
    const groupName = (id: string) => tree.data.find((g) => g.group?.id === id)?.group?.name ?? "";
    const categoryOf = (id: string) => {
      for (const node of tree.data) {
        const found = node.categories.find((c) => c.id === id);
        if (found) return found;
      }
      return null;
    };

    const budgetLine = (status: BudgetStatus, index: number): BudgetLine => {
      const budget = status.budget;
      const label =
        budget == null
          ? ""
          : budget.groupId !== ""
            ? groupName(budget.groupId)
            : (categoryOf(budget.categoryId)?.name ?? "");
      return {
        id: budget?.id ?? `budget-${index}`,
        label,
        spent: fromWire(status.spent, currency),
        limit: fromWire(budget?.limit, currency),
      };
    };

    const memberLine = (spending: MemberSpending, index: number) => ({
      id: spending.member?.userId ?? `member-${index}`,
      name: spending.member?.displayName ?? "",
      spent: fromWire(spending.spent, currency),
    });

    const accountLine = (account: Account) => ({
      id: account.id,
      name: account.name,
      balance: fromWire(account.balance, currency),
    });

    const quickAddPayload = payloadOf(data, WidgetType.QUICK_ADD);
    const quickTemplates =
      quickAddPayload?.case === "quickAdd" ? quickAddPayload.value.templates : templates.data;
    const ownerId = quickTemplates[0]?.memberId ?? "";
    const quickAdd: QuickAddModel = {
      ownerName: memberName(ownerId),
      templates: quickTemplates.slice(0, 3).map((template) => ({
        id: template.id,
        label: template.label,
        icon: template.icon,
        amount: fromWire(template.amount, currency),
      })),
    };

    const monthPayload = payloadOf(data, WidgetType.MONTH);
    const window = periodWindow(period);
    const monthNumber = parseISO(window.from).month;
    const overspentFromList = budgets.data.totalCount - budgets.data.withinLimitCount;
    const month: MonthModel =
      monthPayload?.case === "month"
        ? {
            label: monthPayload.value.label,
            total: fromWire(monthPayload.value.periodTotal, currency),
            slices: home.data.slices.map((slice) => ({
              id: slice.groupId,
              value: fromWire(slice.amount, currency).amountMinor,
            })),
            overspentCount:
              monthPayload.value.budgetCount - monthPayload.value.budgetsWithinLimit,
          }
        : {
            label: monthLabel(monthNumber),
            total: fromWire(home.data.periodTotal, currency),
            slices: home.data.slices.map((slice) => ({
              id: slice.groupId,
              value: fromWire(slice.amount, currency).amountMinor,
            })),
            overspentCount: Math.max(overspentFromList, 0),
          };

    const categoryPayload = payloadOf(data, WidgetType.CATEGORY);
    const topSlice = groupBreakdown.data?.categories[0] ?? null;
    const category: CategoryModel | null =
      categoryPayload?.case === "category"
        ? {
            name: categoryPayload.value.name,
            icon: categoryPayload.value.icon,
            amount: fromWire(categoryPayload.value.amount, currency),
            index: 2,
          }
        : topSlice
          ? {
              name: topSlice.name,
              icon: topSlice.icon,
              amount: fromWire(topSlice.amount, currency),
              index: 2,
            }
          : null;

    const bafPayload = payloadOf(data, WidgetType.BUDGETS_AND_FAMILY);
    const budgetsAndFamily: BudgetsAndFamilyModel =
      bafPayload?.case === "budgetsAndFamily"
        ? {
            budgets: bafPayload.value.budgets.slice(0, 2).map(budgetLine),
            members: bafPayload.value.members.slice(0, 2).map(memberLine),
          }
        : {
            budgets: budgets.data.budgets.slice(0, 2).map(budgetLine),
            members: memberBreakdown.data.members.slice(0, 2).map(memberLine),
          };

    const recentPayload = payloadOf(data, WidgetType.RECENT_TRANSACTIONS);
    const feedTransactions =
      recentPayload?.case === "recentTransactions"
        ? recentPayload.value.transactions
        : feed.data.pages.flatMap((page) => page.days.flatMap((day) => day.transactions));
    const recent: RecentLine[] = feedTransactions.slice(0, 3).map((tx) => {
      const name = categoryOf(tx.categoryId)?.name ?? groupName(tx.groupId);
      const who = memberName(tx.memberId);
      return {
        id: tx.id,
        label: who === "" ? name : `${name} · ${who}`,
        amount: fromWire(tx.amount, currency),
      };
    });

    const accountsPayload = payloadOf(data, WidgetType.ACCOUNTS);
    const shared =
      accountsPayload?.case === "accounts"
        ? accountsPayload.value.accounts.filter((a) => !a.excludedFromFamilyTotal)
        : accounts.data.shared;

    return {
      currency,
      quickAdd,
      month,
      category,
      budgetsAndFamily,
      recent,
      accounts: { accounts: shared.slice(0, 3).map(accountLine) },
    };
  }, [
    home.data,
    budgets.data,
    memberBreakdown.data,
    accounts.data,
    templates.data,
    members.data,
    tree.data,
    feed.data,
    groupBreakdown.data,
    payloads.data,
    period,
    monthLabel,
  ]);

  return {
    previews,
    isPending,
    error,
    refetch: () => {
      for (const query of queries) void query.refetch();
    },
  };
}
