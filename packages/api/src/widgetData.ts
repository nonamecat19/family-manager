import type {
  Account,
  BudgetStatus,
  MemberSpending,
  Transaction,
  WidgetPayload,
  WidgetSize,
  WidgetType,
} from "@fm/sdk/finance/v1/finance_pb";
import type { MealSlot } from "@fm/sdk/recipes/v1/recipes_pb";

import type { Clients } from "./client.ts";
import type { WireMoney } from "./convert.ts";
import { money, type Money } from "./money.ts";

const MAX_LIST_ITEMS = 5;
const MAX_TEXT_LENGTH = 60;
const MAX_WIDGET_IDS = 500;
const MEAL_SLOT_ORDER: Record<number, number> = {
  1: 0,
  2: 1,
  3: 2,
  4: 3,
  5: 4,
};

function fromWire(m: WireMoney | undefined, fallbackCurrency = "EUR"): Money {
  if (!m) return money(0, fallbackCurrency);
  return money(Number(m.amountMinor), m.currencyCode || fallbackCurrency);
}

function truncate(value: string, max = MAX_TEXT_LENGTH): string {
  const chars = Array.from(value);
  return chars.length > max ? `${chars.slice(0, max - 1).join("")}…` : value;
}

function take<T>(items: readonly T[], max = MAX_LIST_ITEMS): T[] {
  return items.slice(0, max);
}

function isoFromTimestamp(
  ts: { seconds: bigint; nanos: number } | undefined,
): string | null {
  if (!ts) return null;
  return new Date(
    Number(ts.seconds) * 1000 + Math.trunc(ts.nanos / 1_000_000),
  ).toISOString();
}

export interface QuickAddTemplateSnapshot {
  id: string;
  memberId: string;
  label: string;
  icon: string;
  amount: Money;
}

export interface BudgetStatusSnapshot {
  budgetId: string;
  memberId: string;
  limit: Money;
  categoryId: string;
  groupId: string;
  spent: Money;
  remaining: Money;
  share: number;
  exceeded: boolean;
  daysRemaining: number;
}

export interface MemberSpendingSnapshot {
  userId: string;
  displayName: string;
  spent: Money;
  share: number;
  transactionCount: number;
}

export interface TransactionSnapshot {
  id: string;
  categoryId: string;
  groupId: string;
  memberId: string;
  type: Transaction["type"];
  merchant: string;
  amount: Money;
  occurredOn: string;
}

export interface AccountSnapshot {
  id: string;
  name: string;
  kind: Account["kind"];
  balance: Money;
}

export type FinanceWidgetSnapshotData =
  | { kind: "quickAdd"; templates: QuickAddTemplateSnapshot[] }
  | {
      kind: "month";
      label: string;
      periodTotal: Money;
      budgetCount: number;
      budgetsWithinLimit: number;
    }
  | {
      kind: "category";
      categoryId: string;
      name: string;
      icon: string;
      amount: Money;
      budget: BudgetStatusSnapshot | null;
    }
  | {
      kind: "budgetsAndFamily";
      budgets: BudgetStatusSnapshot[];
      members: MemberSpendingSnapshot[];
      periodTotal: Money;
    }
  | { kind: "recentTransactions"; transactions: TransactionSnapshot[] }
  | { kind: "accounts"; accounts: AccountSnapshot[]; sharedBalance: Money }
  | { kind: "empty" };

export interface FinanceWidgetSnapshot {
  widgetId: string;
  type: WidgetType;
  size: WidgetSize;
  refreshedAt: string | null;
  data: FinanceWidgetSnapshotData;
}

function budgetStatusSnapshot(status: BudgetStatus): BudgetStatusSnapshot {
  return {
    budgetId: status.budget?.id ?? "",
    memberId: status.budget?.memberId ?? "",
    limit: fromWire(status.budget?.limit),
    categoryId: status.budget?.categoryId ?? "",
    groupId: status.budget?.groupId ?? "",
    spent: fromWire(status.spent),
    remaining: fromWire(status.remaining),
    share: status.share,
    exceeded: status.exceeded,
    daysRemaining: status.daysRemaining,
  };
}

function memberSpendingSnapshot(
  spending: MemberSpending,
): MemberSpendingSnapshot {
  return {
    userId: spending.member?.userId ?? "",
    displayName: truncate(spending.member?.displayName ?? ""),
    spent: fromWire(spending.spent),
    share: spending.share,
    transactionCount: spending.transactionCount,
  };
}

function transactionSnapshot(transaction: Transaction): TransactionSnapshot {
  return {
    id: transaction.id,
    categoryId: transaction.categoryId,
    groupId: transaction.groupId,
    memberId: transaction.memberId,
    type: transaction.type,
    merchant: truncate(transaction.merchant || transaction.note),
    amount: fromWire(transaction.amount),
    occurredOn: transaction.occurredOn,
  };
}

function accountSnapshot(account: Account): AccountSnapshot {
  return {
    id: account.id,
    name: truncate(account.name),
    kind: account.kind,
    balance: fromWire(account.balance),
  };
}

function financeWidgetPayloadData(
  payload: WidgetPayload,
): FinanceWidgetSnapshotData {
  switch (payload.data.case) {
    case "quickAdd":
      return {
        kind: "quickAdd",
        templates: take(payload.data.value.templates).map((template) => ({
          id: template.id,
          memberId: template.memberId,
          label: truncate(template.label),
          icon: template.icon,
          amount: fromWire(template.amount),
        })),
      };
    case "month":
      return {
        kind: "month",
        label: truncate(payload.data.value.label),
        periodTotal: fromWire(payload.data.value.periodTotal),
        budgetCount: payload.data.value.budgetCount,
        budgetsWithinLimit: payload.data.value.budgetsWithinLimit,
      };
    case "category":
      return {
        kind: "category",
        categoryId: payload.data.value.categoryId,
        name: truncate(payload.data.value.name),
        icon: payload.data.value.icon,
        amount: fromWire(payload.data.value.amount),
        budget: payload.data.value.budget
          ? budgetStatusSnapshot(payload.data.value.budget)
          : null,
      };
    case "budgetsAndFamily":
      return {
        kind: "budgetsAndFamily",
        budgets: take(payload.data.value.budgets).map(budgetStatusSnapshot),
        members: take(payload.data.value.members).map(memberSpendingSnapshot),
        periodTotal: fromWire(payload.data.value.periodTotal),
      };
    case "recentTransactions":
      return {
        kind: "recentTransactions",
        transactions: take(payload.data.value.transactions).map(
          transactionSnapshot,
        ),
      };
    case "accounts":
      return {
        kind: "accounts",
        accounts: take(
          payload.data.value.accounts.filter((a) => !a.excludedFromFamilyTotal),
        ).map(accountSnapshot),
        sharedBalance: fromWire(payload.data.value.sharedBalance),
      };
    default:
      return { kind: "empty" };
  }
}

export async function financeWidgetSnapshots(
  clients: Clients,
): Promise<FinanceWidgetSnapshot[]> {
  const { widgets: all } = await clients.finance.listWidgets({});
  const widgets = all.slice(0, MAX_WIDGET_IDS);
  if (widgets.length === 0) return [];

  const { payloads } = await clients.finance.getWidgetData({
    widgetIds: widgets.map((w) => w.id),
  });
  const payloadByWidgetId = new Map(
    payloads.map((payload) => [payload.widgetId, payload]),
  );

  return widgets.map((widget) => {
    const payload = payloadByWidgetId.get(widget.id);
    return {
      widgetId: widget.id,
      type: widget.type,
      size: widget.size,
      refreshedAt: payload ? isoFromTimestamp(payload.refreshedAt) : null,
      data: payload ? financeWidgetPayloadData(payload) : { kind: "empty" },
    };
  });
}

export interface NoteSnapshotItem {
  id: string;
  title: string;
  preview: string;
  starred: boolean;
  taskTotal: number;
  taskDone: number;
  updatedAt: string | null;
}

export interface NoteWidgetSnapshot {
  recent: NoteSnapshotItem[];
  starred: NoteSnapshotItem[];
}

function noteSnapshotItem(note: {
  id: string;
  title: string;
  preview: string;
  starred: boolean;
  taskTotal: number;
  taskDone: number;
  updatedAt?: { seconds: bigint; nanos: number } | undefined;
}): NoteSnapshotItem {
  return {
    id: note.id,
    title: truncate(note.title),
    preview: truncate(note.preview),
    starred: note.starred,
    taskTotal: note.taskTotal,
    taskDone: note.taskDone,
    updatedAt: isoFromTimestamp(note.updatedAt),
  };
}

export async function noteWidgetSnapshot(
  clients: Clients,
): Promise<NoteWidgetSnapshot> {
  const baseFilters = {
    notebookId: "",
    includeArchived: false,
    archivedOnly: false,
    sharedOnly: false,
    pageSize: MAX_LIST_ITEMS,
  };

  const [recentRes, starredRes] = await Promise.all([
    clients.notes.listNotes({ ...baseFilters, starredOnly: false }),
    clients.notes.listNotes({ ...baseFilters, starredOnly: true }),
  ]);

  return {
    recent: take(recentRes.notes).map(noteSnapshotItem),
    starred: take(starredRes.notes).map(noteSnapshotItem),
  };
}

export interface MealPlanItemSnapshot {
  entryId: string;
  recipeId: string;
  title: string;
  slot: MealSlot;
  servings: number;
}

export interface MealPlanWidgetSnapshot {
  date: string;
  entries: MealPlanItemSnapshot[];
}

export async function mealPlanWidgetSnapshot(
  clients: Clients,
  date: string,
): Promise<MealPlanWidgetSnapshot> {
  const { entries } = await clients.recipes.listMealPlan({
    fromDate: date,
    toDate: date,
  });
  const bounded = take(entries);

  const recipeIds = [...new Set(bounded.map((entry) => entry.recipeId))];
  const lookups = await Promise.allSettled(
    recipeIds.map((recipeId) => clients.recipes.getRecipe({ recipeId })),
  );
  const titleByRecipeId = new Map<string, string>();
  lookups.forEach((result, i) => {
    if (result.status === "fulfilled")
      titleByRecipeId.set(recipeIds[i] ?? "", result.value.recipe?.title ?? "");
  });

  return {
    date,
    entries: [...bounded]
      .sort(
        (a, b) =>
          (MEAL_SLOT_ORDER[a.slot] ?? 9) - (MEAL_SLOT_ORDER[b.slot] ?? 9),
      )
      .map((entry) => ({
        entryId: entry.id,
        recipeId: entry.recipeId,
        title: truncate(titleByRecipeId.get(entry.recipeId) ?? ""),
        slot: entry.slot,
        servings: entry.servings,
      })),
  };
}
