import type { DateRange } from "./dates.ts";

/**
 * Query keys, in one place. Every key starts with a domain segment so a mutation can
 * invalidate a whole domain (`["finance"]`) without knowing every hook that reads it.
 */
export const queryKeys = {
  finance: ["finance"] as const,

  accounts: (includeArchived = false) => ["finance", "accounts", { includeArchived }] as const,
  account: (id: string) => ["finance", "accounts", id] as const,

  categories: (kind?: number, includeArchived = false) =>
    ["finance", "categories", { kind: kind ?? 0, includeArchived }] as const,

  transactions: (filters: TransactionFilters) => ["finance", "transactions", filters] as const,
  transaction: (id: string) => ["finance", "transactions", id] as const,

  summary: (range: DateRange, accountIds: readonly string[] = []) =>
    ["finance", "summary", range, [...accountIds].sort()] as const,

  breakdown: (range: DateRange, type: number, accountIds: readonly string[] = []) =>
    ["finance", "breakdown", range, type, [...accountIds].sort()] as const,

  // asOf is part of the key: the same budget has different progress in different windows.
  budgets: (asOf = "", includeArchived = false) =>
    ["finance", "budgets", { asOf, includeArchived }] as const,
  budget: (id: string, asOf = "") => ["finance", "budgets", id, { asOf }] as const,

  family: ["family"] as const,
  familyDetail: () => ["family", "detail"] as const,
  members: () => ["family", "members"] as const,
  invitations: () => ["family", "invitations"] as const,
} as const;

export interface TransactionFilters {
  range: DateRange;
  accountIds?: readonly string[];
  categoryIds?: readonly string[];
  /** finance.v1.TransactionType; 0 means "any". */
  type?: number;
  search?: string;
}

/**
 * Normalises a filter object so two equivalent filters produce the same cache key — array
 * order and undefined-vs-empty must not fragment the cache.
 */
export function normalizeFilters(f: TransactionFilters): TransactionFilters {
  return {
    range: f.range,
    accountIds: [...(f.accountIds ?? [])].sort(),
    categoryIds: [...(f.categoryIds ?? [])].sort(),
    type: f.type ?? 0,
    search: (f.search ?? "").trim(),
  };
}
