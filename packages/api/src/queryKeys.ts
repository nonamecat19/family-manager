import type { NoteListFilters } from "./notes.ts";
import type { RecipeListFilters } from "./recipes.ts";
import {
  periodKey,
  scopeKey,
  type PeriodInput,
  type PeriodWindowOptions,
  type ScopeInput,
} from "./scope.ts";

/**
 * Query keys, in one place. Every key starts with a domain segment so a mutation can
 * invalidate a whole domain (`["recipes"]`, `["finance"]`) without knowing every hook that
 * reads it. Finance keys carry a second segment per entity as well, so a template tap can
 * repaint just the chips before the domain-wide invalidation lands.
 */

export interface TransactionFilters {
  scope?: ScopeInput;
  period?: PeriodInput;
  /** 0 = both (the ЗАГАЛЬНЕ tab), 1 = expense, 2 = income — finance.v1.TransactionKind. */
  kind?: number;
  memberIds?: readonly string[];
  accountIds?: readonly string[];
  groupIds?: readonly string[];
  categoryIds?: readonly string[];
  /** Matches note and merchant, case-insensitively. */
  query?: string;
  pageSize?: number;
}

/**
 * Filters normalized to one canonical shape: arrays sorted and copied, blanks defaulted.
 * Two filter objects that mean the same thing must key the same cache entry, or the feed
 * refetches every time a screen rebuilds its filter object in a different order.
 */
export function normalizeTransactionFilters(
  filters: TransactionFilters = {},
  opts: PeriodWindowOptions = {},
) {
  return {
    scope: scopeKey(filters.scope),
    period: filters.period ? periodKey(filters.period, opts) : "",
    kind: filters.kind ?? 0,
    memberIds: sorted(filters.memberIds),
    accountIds: sorted(filters.accountIds),
    groupIds: sorted(filters.groupIds),
    categoryIds: sorted(filters.categoryIds),
    query: (filters.query ?? "").trim().toLowerCase(),
    pageSize: filters.pageSize ?? 0,
  } as const;
}

function sorted(values: readonly string[] | undefined): string[] {
  return values ? [...values].sort() : [];
}

/**
 * Filters normalized to one canonical shape, for the same reason the transaction filters are:
 * the notes list pane rebuilds its filter object on every render, and an object that differs
 * only in which optional keys were spelled out must not be a second cache entry — that is a
 * refetch of the whole list every time the sidebar re-renders.
 *
 * `sort` and `facet`-like enum fields normalize to their proto zero value rather than to
 * undefined, so "not set" and "explicitly the default" key the same entry.
 */
export function normalizeNoteFilters(filters: NoteListFilters = {}) {
  return {
    notebookId: filters.notebookId ?? "",
    starredOnly: filters.starredOnly ?? false,
    includeArchived: filters.includeArchived ?? false,
    archivedOnly: filters.archivedOnly ?? false,
    sharedOnly: filters.sharedOnly ?? false,
    sort: filters.sort ?? 0,
    pageSize: filters.pageSize ?? 0,
  } as const;
}

export interface CategoryTreeFilters {
  kind?: number;
  includeArchived?: boolean;
  /** Which budget window the group statuses are evaluated in; empty means today. */
  asOf?: string;
}

export interface BudgetFilters {
  asOf?: string;
  /** finance.v1.BudgetTargetFilter: 0 = all, 1 = group budgets, 2 = category budgets. */
  target?: number;
  includeArchived?: boolean;
}

export const queryKeys = {
  family: ["family"] as const,
  familyDetail: () => ["family", "detail"] as const,
  members: () => ["family", "members"] as const,
  invitations: () => ["family", "invitations"] as const,

  recipes: ["recipes"] as const,
  recipeCategories: () => ["recipes", "categories"] as const,
  recipeSubcategories: (categoryId: string) => ["recipes", "categories", categoryId, "subcategories"] as const,
  recipesList: (filters: RecipeListFilters) => ["recipes", "list", filters] as const,
  recipe: (id: string) => ["recipes", "detail", id] as const,
  favoriteRecipes: () => ["recipes", "favorites"] as const,
  recipeComments: (recipeId: string) => ["recipes", "comments", recipeId] as const,
  mealPlan: (fromDate: string, toDate: string) => ["recipes", "mealPlan", fromDate, toDate] as const,
  totalIngredients: (fromDate: string, toDate: string) =>
    ["recipes", "totalIngredients", fromDate, toDate] as const,
  sumIngredients: (items: readonly { recipeId: string; servings: number }[]) =>
    ["recipes", "sumIngredients", items] as const,

  /* -------------------------------------------------------------------- notes */

  // The notes domain root. A note edit moves the list row, the notebook's count and the
  // activity rail at once, so mutations invalidate this rather than naming each reader.
  notes: ["notes"] as const,
  notesList: (filters: NoteListFilters = {}) => ["notes", "list", normalizeNoteFilters(filters)] as const,
  note: (id: string) => ["notes", "detail", id] as const,
  notebooks: (includeArchived = false) => ["notes", "notebooks", includeArchived] as const,
  // The query is trimmed and lowercased here: the palette fires a key at a time and two
  // spellings of the same search must not be two round trips.
  noteSearch: (query: string, facet = 0) =>
    ["notes", "search", query.trim().toLowerCase(), facet] as const,
  // Shares hang off either a note or a notebook, and the two id spaces are separate — the
  // kind is part of the key so a notebook id can never read a note's share list.
  noteShares: (id: string, kind: "note" | "notebook" = "note") => ["notes", "shares", kind, id] as const,
  noteComments: (id: string, includeResolved = false) =>
    ["notes", "comments", id, includeResolved] as const,
  noteActivity: (id: string) => ["notes", "activity", id] as const,
  sharedWithMe: () => ["notes", "sharedWithMe"] as const,

  /* ------------------------------------------------------------------ finance */

  finance: ["finance"] as const,

  financeSettings: () => ["finance", "settings"] as const,
  financeOverview: (period: PeriodInput) => ["finance", "overview", periodKey(period)] as const,
  financeMembers: (includePending = false) => ["finance", "members", includePending] as const,

  financeAccounts: ["finance", "accounts"] as const,
  financeAccountsList: (includeArchived = false) =>
    ["finance", "accounts", "list", includeArchived] as const,
  financeAccount: (accountId: string) => ["finance", "accounts", "detail", accountId] as const,

  financeCategories: ["finance", "categories"] as const,
  financeCategoryTree: (filters: CategoryTreeFilters = {}) =>
    [
      "finance",
      "categories",
      "tree",
      { kind: filters.kind ?? 0, includeArchived: filters.includeArchived ?? false, asOf: filters.asOf ?? "" },
    ] as const,

  financeTransactions: ["finance", "transactions"] as const,
  financeTransactionsList: (filters: TransactionFilters = {}) =>
    ["finance", "transactions", "list", normalizeTransactionFilters(filters)] as const,
  financeTransactionFeed: (filters: TransactionFilters = {}) =>
    ["finance", "transactions", "feed", normalizeTransactionFilters(filters)] as const,
  financeTransaction: (transactionId: string) =>
    ["finance", "transactions", "detail", transactionId] as const,

  financeTemplates: ["finance", "templates"] as const,
  financeTemplatesList: (ownerUserId = "") => ["finance", "templates", "list", ownerUserId] as const,

  financeBudgets: ["finance", "budgets"] as const,
  financeBudgetsList: (filters: BudgetFilters = {}) =>
    [
      "finance",
      "budgets",
      "list",
      { asOf: filters.asOf ?? "", target: filters.target ?? 0, includeArchived: filters.includeArchived ?? false },
    ] as const,

  financeAnalytics: ["finance", "analytics"] as const,
  financeHomeSummary: (scope: ScopeInput, period: PeriodInput, kind = 0) =>
    ["finance", "analytics", "home", scopeKey(scope), periodKey(period), kind] as const,
  financeGroupBreakdown: (groupId: string, scope: ScopeInput, period: PeriodInput, kind = 0) =>
    ["finance", "analytics", "group", groupId, scopeKey(scope), periodKey(period), kind] as const,
  financeMemberBreakdown: (period: PeriodInput, kind = 0) =>
    ["finance", "analytics", "members", periodKey(period), kind] as const,
  financeSpendingSeries: (params: {
    granularity: string;
    bucketCount: number;
    kind?: number;
    stackedBy?: number;
    scope?: ScopeInput;
  }) =>
    [
      "finance",
      "analytics",
      "series",
      {
        granularity: params.granularity,
        bucketCount: params.bucketCount,
        kind: params.kind ?? 0,
        stackedBy: params.stackedBy ?? 0,
        scope: scopeKey(params.scope),
      },
    ] as const,
  financeInsights: (period: PeriodInput, limit = 0) =>
    ["finance", "analytics", "insights", periodKey(period), limit] as const,

  financeRecurring: ["finance", "recurring"] as const,
  financeRecurringList: (includeInactive = false, asOf = "") =>
    ["finance", "recurring", "list", includeInactive, asOf] as const,

  financeReminders: ["finance", "reminders"] as const,
  financeRemindersList: (includeDisabled = false) =>
    ["finance", "reminders", "list", includeDisabled] as const,

  financeWidgets: ["finance", "widgets"] as const,
  financeWidgetsList: () => ["finance", "widgets", "list"] as const,
  // Widget ids are sorted: the same placement set requested in a different order is the same
  // refresh, and a home-screen widget process rebuilds that array on every wake.
  financeWidgetData: (widgetIds: readonly string[] = []) =>
    ["finance", "widgets", "data", [...widgetIds].sort()] as const,
} as const;
