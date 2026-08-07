import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  AccountType,
  BudgetPeriod,
  Transaction,
  TransactionType,
} from "@fm/sdk/finance/v1/finance_pb";

import { useClients } from "./provider.tsx";
import { queryKeys, normalizeFilters, type TransactionFilters } from "./queryKeys.ts";
import { toWire } from "./convert.ts";
import type { DateRange } from "./dates.ts";
import type { Money } from "./money.ts";

/* ------------------------------------------------------------------ accounts */

export function useAccounts(includeArchived = false) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.accounts(includeArchived),
    queryFn: () => finance.listAccounts({ includeArchived }),
  });
}

export function useAccount(id: string) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.account(id),
    queryFn: () => finance.getAccount({ id }),
    enabled: id !== "",
  });
}

export interface CreateAccountInput {
  name: string;
  type: AccountType;
  currencyCode: string;
  openingBalance: Money;
  color: string;
  icon: string;
}

export function useCreateAccount() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateAccountInput) =>
      finance.createAccount({ ...input, openingBalance: toWire(input.openingBalance) }),
    // An account changes balances and totals everywhere; invalidate the domain, not one key.
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export interface UpdateAccountInput {
  id: string;
  name: string;
  type: AccountType;
  color?: string;
  icon?: string;
  archived?: boolean;
  sortOrder?: number;
}

export function useUpdateAccount() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateAccountInput) => finance.updateAccount(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteAccount() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => finance.deleteAccount({ id }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

/* ---------------------------------------------------------------- categories */

export function useCategories(kind?: TransactionType, includeArchived = false) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.categories(kind, includeArchived),
    queryFn: () => finance.listCategories({ kind, includeArchived }),
  });
}

export function useCreateCategory() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: {
      name: string;
      kind: TransactionType;
      color: string;
      icon: string;
      parentId?: string;
    }) => finance.createCategory(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export interface UpdateCategoryInput {
  id: string;
  name: string;
  color?: string;
  icon?: string;
  parentId?: string;
  archived?: boolean;
  sortOrder?: number;
}

export function useUpdateCategory() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateCategoryInput) => finance.updateCategory(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteCategory() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => finance.deleteCategory({ id }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

/* -------------------------------------------------------------- transactions */

/** Paged ledger. The list screen is the app's home, so it pages rather than loading a year. */
export function useTransactions(filters: TransactionFilters, pageSize = 50) {
  const { finance } = useClients();
  const normalized = normalizeFilters(filters);

  return useInfiniteQuery({
    queryKey: queryKeys.transactions(normalized),
    initialPageParam: "",
    queryFn: ({ pageParam }) =>
      finance.listTransactions({
        range: normalized.range,
        accountIds: [...(normalized.accountIds ?? [])],
        categoryIds: [...(normalized.categoryIds ?? [])],
        type: normalized.type as TransactionType,
        search: normalized.search,
        pageSize,
        pageToken: pageParam,
      }),
    getNextPageParam: (last) => (last.nextPageToken === "" ? undefined : last.nextPageToken),
  });
}

export interface TransactionInput {
  accountId: string;
  counterAccountId?: string;
  categoryId: string;
  type: TransactionType;
  amount: Money;
  note?: string;
  /** YYYY-MM-DD. */
  occurredOn: string;
}

export function useCreateTransaction() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: TransactionInput) =>
      finance.createTransaction({ ...input, amount: toWire(input.amount) }),
    // Balances, summaries and the pie all move when a transaction lands.
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useUpdateTransaction() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: TransactionInput & { id: string }) =>
      finance.updateTransaction({ ...input, amount: toWire(input.amount) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteTransaction() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => finance.deleteTransaction({ id }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export type { Transaction };

/* ------------------------------------------------------------------- reports */

export function useSummary(range: DateRange, accountIds: readonly string[] = []) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.summary(range, accountIds),
    queryFn: () => finance.getSummary({ range, accountIds: [...accountIds] }),
  });
}

export function useCategoryBreakdown(
  range: DateRange,
  type: TransactionType,
  accountIds: readonly string[] = [],
) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.breakdown(range, type, accountIds),
    queryFn: () => finance.getCategoryBreakdown({ range, type, accountIds: [...accountIds] }),
  });
}

/* -------------------------------------------------------------------- family */

export function useFamily() {
  const { family } = useClients();
  return useQuery({
    queryKey: queryKeys.familyDetail(),
    queryFn: () => family.getFamily({}),
  });
}

export function useMembers(familyId: string) {
  const { family } = useClients();
  return useQuery({
    queryKey: queryKeys.members(),
    queryFn: () => family.listMembers({ familyId }),
    enabled: familyId !== "",
  });
}

export function useCreateFamily() {
  const { family } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => family.createFamily({ name }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.family }),
  });
}

export function useInviteMember() {
  const { family } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { familyId: string; email: string }) => family.inviteMember(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.family }),
  });
}

export function useAcceptInvitation() {
  const { family } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (token: string) => family.acceptInvitation({ token }),
    // Joining a family changes which ledger the user sees: drop everything.
    onSuccess: () => qc.invalidateQueries(),
  });
}

/* ------------------------------------------------------------------- budgets */

export function useBudgets(asOf = "", includeArchived = false) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.budgets(asOf, includeArchived),
    queryFn: () => finance.listBudgets({ asOf, includeArchived }),
  });
}

export function useBudget(id: string, asOf = "") {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.budget(id, asOf),
    queryFn: () => finance.getBudget({ id, asOf }),
    enabled: id !== "",
  });
}

export interface BudgetInput {
  name: string;
  /** Empty means the budget covers every expense in the household. */
  categoryId?: string;
  limit: Money;
  period: BudgetPeriod;
  /** YYYY-MM-DD; anchors the recurring window. */
  startOn: string;
}

export function useCreateBudget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: BudgetInput) =>
      finance.createBudget({ ...input, limit: toWire(input.limit) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useUpdateBudget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: BudgetInput & { id: string; archived?: boolean; sortOrder?: number }) =>
      finance.updateBudget({ ...input, limit: toWire(input.limit) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteBudget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => finance.deleteBudget({ id }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}
