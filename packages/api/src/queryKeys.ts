import type { NoteListFilters } from "./notes.ts";
import type { RecipeListFilters } from "./recipes.ts";
import {
  periodKey,
  scopeKey,
  type PeriodInput,
  type PeriodWindowOptions,
  type ScopeInput,
} from "./scope.ts";


export interface TransactionFilters {
  scope?: ScopeInput;
  period?: PeriodInput;
  kind?: number;
  memberIds?: readonly string[];
  accountIds?: readonly string[];
  groupIds?: readonly string[];
  categoryIds?: readonly string[];
  query?: string;
  pageSize?: number;
}

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
  asOf?: string;
}

export interface BudgetFilters {
  asOf?: string;
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


  notes: ["notes"] as const,
  notesList: (filters: NoteListFilters = {}) => ["notes", "list", normalizeNoteFilters(filters)] as const,
  note: (id: string) => ["notes", "detail", id] as const,
  notebooks: (includeArchived = false) => ["notes", "notebooks", includeArchived] as const,
  noteSearch: (query: string, facet = 0) =>
    ["notes", "search", query.trim().toLowerCase(), facet] as const,
  noteShares: (id: string, kind: "note" | "notebook" = "note") => ["notes", "shares", kind, id] as const,
  noteComments: (id: string, includeResolved = false) =>
    ["notes", "comments", id, includeResolved] as const,
  noteActivity: (id: string) => ["notes", "activity", id] as const,
  sharedWithMe: () => ["notes", "sharedWithMe"] as const,


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
  financeWidgetData: (widgetIds: readonly string[] = []) =>
    ["finance", "widgets", "data", [...widgetIds].sort()] as const,
} as const;
