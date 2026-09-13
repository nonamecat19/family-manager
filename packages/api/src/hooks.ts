import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  AccountKind,
  AccountVisibility,
  BudgetPeriod,
  BudgetTargetFilter,
  BudgetTargetKind,
  InsightKind,
  MemberRole,
  MemberStatus,
  RecurrenceUnit,
  ReminderKind,
  ScopeKind,
  SeriesStacking,
  TransactionKind,
  TransactionType,
  Weekday,
  WidgetSize,
  WidgetType,
  type QuickTemplate,
} from "@fm/sdk/finance/v1/finance_pb";

import {
  Role as FamilyMemberRole,
  InvitationStatus,
  type Family,
  type Invitation,
  type Member as FamilyMember,
} from "@fm/sdk/family/v1/family_pb";

import { toWire } from "./convert.ts";
import type { Money } from "./money.ts";
import { bumpTemplateUsage } from "./optimistic.ts";
import { useClients } from "./provider.tsx";
import {
  queryKeys,
  type BudgetFilters,
  type CategoryTreeFilters,
  type TransactionFilters,
} from "./queryKeys.ts";
import { familyScope, type PeriodGranularityInput, type PeriodInput, type ScopeInput } from "./scope.ts";
import { toWireGranularity, toWirePeriod, toWireScope } from "./scopeWire.ts";



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

export function useUpdateFamily() {
  const { family } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { familyId: string; name: string }) => family.updateFamily(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.family }),
  });
}

export function useInvitations(familyId: string, opts: { enabled?: boolean } = {}) {
  const { family } = useClients();
  return useQuery({
    queryKey: queryKeys.invitations(),
    queryFn: () => family.listInvitations({ familyId }),
    enabled: familyId !== "" && (opts.enabled ?? true),
  });
}

export function useRevokeInvitation() {
  const { family } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (invitationId: string) => family.revokeInvitation({ invitationId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.family }),
  });
}

export function useRemoveMember() {
  const { family } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { familyId: string; userId: string }) => family.removeMember(input),
    onSuccess: () => qc.invalidateQueries(),
  });
}

export function useLeaveFamily() {
  const { family } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (familyId: string) => family.leaveFamily({ familyId }),
    onSuccess: () => qc.invalidateQueries(),
  });
}

export function useAcceptInvitation() {
  const { family } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (token: string) => family.acceptInvitation({ token }),
    onSuccess: () => qc.invalidateQueries(),
  });
}


export interface BootstrapHouseholdInput {
  baseCurrencyCode: string;
  timezone: string;
  seedDefaultTaxonomy?: boolean;
  weekStartsOn?: Weekday;
}

export function useBootstrapHousehold() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: BootstrapHouseholdInput) =>
      finance.bootstrapHousehold({
        baseCurrencyCode: input.baseCurrencyCode,
        timezone: input.timezone,
        seedDefaultTaxonomy: input.seedDefaultTaxonomy ?? true,
        weekStartsOn: input.weekStartsOn ?? Weekday.MONDAY,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useHouseholdOverview(period: PeriodInput) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeOverview(period),
    queryFn: () => finance.getHouseholdOverview({ period: toWirePeriod(period) }),
  });
}

export function useFinanceSettings() {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeSettings(),
    queryFn: async () => (await finance.getFinanceSettings({})).settings ?? null,
  });
}

export interface UpdateFinanceSettingsInput {
  baseCurrencyCode?: string;
  weekStartsOn?: Weekday;
  overspendNotificationsEnabled?: boolean;
  pinLockEnabled?: boolean;
  timezone?: string;
}

export function useUpdateFinanceSettings() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateFinanceSettingsInput) =>
      (await finance.updateFinanceSettings(input)).settings ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useSetOverspendNotifications() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (enabled: boolean) =>
      (await finance.setOverspendNotifications({ enabled })).enabled,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useFinanceMembers(includePending = false) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeMembers(includePending),
    queryFn: async () => (await finance.listMembers({ includePending })).members,
  });
}


export function useAccounts(includeArchived = false) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeAccountsList(includeArchived),
    queryFn: () => finance.listAccounts({ includeArchived }),
  });
}

export function useAccount(accountId: string) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeAccount(accountId),
    queryFn: async () => (await finance.getAccount({ accountId })).account ?? null,
    enabled: accountId !== "",
  });
}

export interface CreateAccountInput {
  name: string;
  kind: AccountKind;
  visibility: AccountVisibility;
  currencyCode: string;
  openingBalance: Money;
  icon?: string;
  colorStep?: number;
  excludedFromFamilyTotal?: boolean;
}

export function useCreateAccount() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateAccountInput) =>
      (
        await finance.createAccount({
          name: input.name,
          kind: input.kind,
          visibility: input.visibility,
          currencyCode: input.currencyCode,
          openingBalance: toWire(input.openingBalance),
          icon: input.icon ?? "",
          colorStep: input.colorStep ?? 0,
          excludedFromFamilyTotal: input.excludedFromFamilyTotal ?? false,
        })
      ).account ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export interface UpdateAccountInput {
  accountId: string;
  name?: string;
  kind?: AccountKind;
  visibility?: AccountVisibility;
  icon?: string;
  colorStep?: number;
  excludedFromFamilyTotal?: boolean;
  openingBalance?: Money;
}

export function useUpdateAccount() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ openingBalance, ...rest }: UpdateAccountInput) =>
      (
        await finance.updateAccount({
          ...rest,
          openingBalance: openingBalance ? toWire(openingBalance) : undefined,
        })
      ).account ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useArchiveAccount() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { accountId: string; archived: boolean }) =>
      (await finance.archiveAccount(input)).account ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteAccount() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (accountId: string) => finance.deleteAccount({ accountId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useReorderAccounts() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (accountIdsInOrder: string[]) => finance.reorderAccounts({ accountIdsInOrder }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeAccounts }),
  });
}

export interface TransferInput {
  fromAccountId: string;
  toAccountId: string;
  amount: Money;
  receivedAmount?: Money;
  occurredOn: string;
  note?: string;
  memberId?: string;
}

export function useTransferBetweenAccounts() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: TransferInput) =>
      (
        await finance.transferBetweenAccounts({
          fromAccountId: input.fromAccountId,
          toAccountId: input.toAccountId,
          amount: toWire(input.amount),
          receivedAmount: input.receivedAmount ? toWire(input.receivedAmount) : undefined,
          occurredOn: input.occurredOn,
          note: input.note ?? "",
          memberId: input.memberId ?? "",
        })
      ).transaction ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}


export function useCategoryTree(filters: CategoryTreeFilters = {}) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeCategoryTree(filters),
    queryFn: async () =>
      (
        await finance.listCategoryTree({
          kind: filters.kind ?? TransactionKind.UNSPECIFIED,
          includeArchived: filters.includeArchived ?? false,
          asOf: filters.asOf ?? "",
        })
      ).groups,
  });
}

export interface CreateCategoryGroupInput {
  name: string;
  kind: TransactionKind;
  icon?: string;
  colorStep?: number;
}

export function useCreateCategoryGroup() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateCategoryGroupInput) =>
      (
        await finance.createCategoryGroup({
          name: input.name,
          kind: input.kind,
          icon: input.icon ?? "",
          colorStep: input.colorStep ?? 0,
        })
      ).group ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export interface UpdateCategoryGroupInput {
  groupId: string;
  name?: string;
  icon?: string;
  colorStep?: number;
  archived?: boolean;
}

export function useUpdateCategoryGroup() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateCategoryGroupInput) =>
      (await finance.updateCategoryGroup(input)).group ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteCategoryGroup() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { groupId: string; reassignToGroupId?: string }) =>
      finance.deleteCategoryGroup({
        groupId: input.groupId,
        reassignToGroupId: input.reassignToGroupId ?? "",
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useReorderCategoryGroups() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (groupIdsInOrder: string[]) => finance.reorderCategoryGroups({ groupIdsInOrder }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeCategories }),
  });
}

export interface CreateCategoryInput {
  groupId: string;
  name: string;
  icon?: string;
  kind: TransactionKind;
}

export function useCreateCategory() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateCategoryInput) =>
      (
        await finance.createCategory({
          groupId: input.groupId,
          name: input.name,
          icon: input.icon ?? "",
          kind: input.kind,
        })
      ).category ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export interface UpdateCategoryInput {
  categoryId: string;
  name?: string;
  icon?: string;
  archived?: boolean;
}

export function useUpdateCategory() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateCategoryInput) =>
      (await finance.updateCategory(input)).category ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useMoveCategory() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { categoryId: string; targetGroupId: string }) =>
      (await finance.moveCategory(input)).category ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteCategory() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { categoryId: string; reassignToCategoryId?: string }) =>
      finance.deleteCategory({
        categoryId: input.categoryId,
        reassignToCategoryId: input.reassignToCategoryId ?? "",
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useReorderCategories() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { groupId: string; categoryIdsInOrder: string[] }) =>
      finance.reorderCategories(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeCategories }),
  });
}


function transactionRequest(filters: TransactionFilters, cursor = "") {
  return {
    scope: toWireScope(filters.scope ?? familyScope),
    period: filters.period ? toWirePeriod(filters.period) : undefined,
    kind: filters.kind ?? TransactionKind.UNSPECIFIED,
    memberIds: [...(filters.memberIds ?? [])],
    accountIds: [...(filters.accountIds ?? [])],
    groupIds: [...(filters.groupIds ?? [])],
    categoryIds: [...(filters.categoryIds ?? [])],
    query: filters.query ?? "",
    cursor,
    pageSize: filters.pageSize ?? 0,
  };
}

export function useTransactions(filters: TransactionFilters = {}, opts: { enabled?: boolean } = {}) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeTransactionsList(filters),
    queryFn: () => finance.listTransactions(transactionRequest(filters)),
    enabled: opts.enabled ?? true,
  });
}

export function useTransactionFeed(filters: TransactionFilters = {}, opts: { enabled?: boolean } = {}) {
  const { finance } = useClients();
  return useInfiniteQuery({
    queryKey: queryKeys.financeTransactionFeed(filters),
    queryFn: ({ pageParam }) => finance.listTransactions(transactionRequest(filters, pageParam)),
    initialPageParam: "",
    getNextPageParam: (last) => (last.nextCursor === "" ? undefined : last.nextCursor),
    enabled: opts.enabled ?? true,
  });
}

export function useTransaction(transactionId: string) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeTransaction(transactionId),
    queryFn: async () => (await finance.getTransaction({ transactionId })).transaction ?? null,
    enabled: transactionId !== "",
  });
}

export interface CreateTransactionInput {
  type: TransactionType;
  accountId: string;
  categoryId: string;
  amount: Money;
  occurredOn: string;
  note?: string;
  merchant?: string;
  memberId?: string;
  templateId?: string;
}

export function useCreateTransaction() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTransactionInput) =>
      finance.createTransaction({
        type: input.type,
        accountId: input.accountId,
        categoryId: input.categoryId,
        amount: toWire(input.amount),
        occurredOn: input.occurredOn,
        note: input.note ?? "",
        merchant: input.merchant ?? "",
        memberId: input.memberId ?? "",
        templateId: input.templateId ?? "",
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export interface UpdateTransactionInput {
  transactionId: string;
  type?: TransactionType;
  accountId?: string;
  categoryId?: string;
  occurredOn?: string;
  note?: string;
  merchant?: string;
  memberId?: string;
  amount?: Money;
}

export function useUpdateTransaction() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ amount, ...rest }: UpdateTransactionInput) =>
      finance.updateTransaction({ ...rest, amount: amount ? toWire(amount) : undefined }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteTransaction() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (transactionId: string) => finance.deleteTransaction({ transactionId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}


export function useTemplates(ownerUserId = "") {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeTemplatesList(ownerUserId),
    queryFn: async () => (await finance.listTemplates({ ownerUserId })).templates,
  });
}

export interface CreateTemplateInput {
  label: string;
  icon?: string;
  amount: Money;
  type: TransactionType;
  categoryId: string;
  accountId: string;
  memberId?: string;
}

export function useCreateTemplate() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateTemplateInput) =>
      (
        await finance.createTemplate({
          label: input.label,
          icon: input.icon ?? "",
          amount: toWire(input.amount),
          type: input.type,
          categoryId: input.categoryId,
          accountId: input.accountId,
          memberId: input.memberId ?? "",
        })
      ).template ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeTemplates }),
  });
}

export interface UpdateTemplateInput {
  templateId: string;
  label?: string;
  icon?: string;
  categoryId?: string;
  accountId?: string;
  memberId?: string;
  amount?: Money;
}

export function useUpdateTemplate() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ amount, ...rest }: UpdateTemplateInput) =>
      (await finance.updateTemplate({ ...rest, amount: amount ? toWire(amount) : undefined }))
        .template ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeTemplates }),
  });
}

export function useDeleteTemplate() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (templateId: string) => finance.deleteTemplate({ templateId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeTemplates }),
  });
}

export function useReorderTemplates() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (templateIdsInOrder: string[]) => finance.reorderTemplates({ templateIdsInOrder }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeTemplates }),
  });
}

export interface LogTemplateInput {
  templateId: string;
  occurredOn?: string;
  amountOverride?: Money;
}

export function useLogTemplate() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: LogTemplateInput) =>
      finance.logTemplate({
        templateId: input.templateId,
        occurredOn: input.occurredOn ?? "",
        amountOverride: input.amountOverride ? toWire(input.amountOverride) : undefined,
      }),
    onMutate: async (input: LogTemplateInput) => {
      await qc.cancelQueries({ queryKey: queryKeys.financeTemplates });
      const previous = qc.getQueriesData<readonly QuickTemplate[]>({
        queryKey: queryKeys.financeTemplates,
      });
      qc.setQueriesData<readonly QuickTemplate[]>({ queryKey: queryKeys.financeTemplates }, (list) =>
        bumpTemplateUsage(list, input.templateId),
      );
      return { previous };
    },
    onError: (_error, _input, context) => {
      for (const [key, data] of context?.previous ?? []) qc.setQueryData(key, data);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}


export function useBudgets(filters: BudgetFilters = {}) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeBudgetsList(filters),
    queryFn: () =>
      finance.listBudgets({
        asOf: filters.asOf ?? "",
        target: filters.target ?? BudgetTargetFilter.UNSPECIFIED,
        includeArchived: filters.includeArchived ?? false,
      }),
  });
}

export interface CreateBudgetInput {
  groupId?: string;
  categoryId?: string;
  limit: Money;
  period: BudgetPeriod;
  startOn: string;
  memberId?: string;
  notifyOnExceed?: boolean;
}

export function useCreateBudget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateBudgetInput) => {
      const target = input.groupId
        ? ({ case: "groupId" as const, value: input.groupId })
        : ({ case: "categoryId" as const, value: input.categoryId ?? "" });
      const res = await finance.createBudget({
        target,
        limit: toWire(input.limit),
        period: input.period,
        startOn: input.startOn,
        memberId: input.memberId ?? "",
        notifyOnExceed: input.notifyOnExceed ?? true,
      });
      return res.budget ?? null;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export interface UpdateBudgetInput {
  budgetId: string;
  limit?: Money;
  period?: BudgetPeriod;
  startOn?: string;
  notifyOnExceed?: boolean;
  archived?: boolean;
}

export function useUpdateBudget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ limit, ...rest }: UpdateBudgetInput) =>
      (await finance.updateBudget({ ...rest, limit: limit ? toWire(limit) : undefined })).budget ??
      null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useDeleteBudget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (budgetId: string) => finance.deleteBudget({ budgetId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}


export interface AggregateParams {
  scope?: ScopeInput;
  period: PeriodInput;
  kind?: TransactionKind;
}

export function useHomeSummary(params: AggregateParams) {
  const { finance } = useClients();
  const scope = params.scope ?? familyScope;
  const kind = params.kind ?? TransactionKind.UNSPECIFIED;
  return useQuery({
    queryKey: queryKeys.financeHomeSummary(scope, params.period, kind),
    queryFn: () =>
      finance.getHomeSummary({
        scope: toWireScope(scope),
        period: toWirePeriod(params.period),
        kind,
      }),
  });
}

export function useGroupBreakdown(groupId: string, params: AggregateParams) {
  const { finance } = useClients();
  const scope = params.scope ?? familyScope;
  const kind = params.kind ?? TransactionKind.UNSPECIFIED;
  return useQuery({
    queryKey: queryKeys.financeGroupBreakdown(groupId, scope, params.period, kind),
    queryFn: () =>
      finance.getGroupBreakdown({
        groupId,
        scope: toWireScope(scope),
        period: toWirePeriod(params.period),
        kind,
      }),
    enabled: groupId !== "",
  });
}

export function useMemberBreakdown(period: PeriodInput, kind: TransactionKind = TransactionKind.UNSPECIFIED) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeMemberBreakdown(period, kind),
    queryFn: () => finance.getMemberBreakdown({ period: toWirePeriod(period), kind }),
  });
}

export interface SpendingSeriesParams {
  granularity: PeriodGranularityInput;
  bucketCount: number;
  kind?: TransactionKind;
  stackedBy?: SeriesStacking;
  scope?: ScopeInput;
}

export function useSpendingSeries(params: SpendingSeriesParams) {
  const { finance } = useClients();
  const kind = params.kind ?? TransactionKind.UNSPECIFIED;
  const stackedBy = params.stackedBy ?? SeriesStacking.NONE;
  const scope = params.scope ?? familyScope;
  return useQuery({
    queryKey: queryKeys.financeSpendingSeries({
      granularity: params.granularity,
      bucketCount: params.bucketCount,
      kind,
      stackedBy,
      scope,
    }),
    queryFn: () =>
      finance.getSpendingSeries({
        granularity: toWireGranularity(params.granularity),
        bucketCount: params.bucketCount,
        kind,
        stackedBy,
        scope: toWireScope(scope),
      }),
  });
}

export function useInsights(period: PeriodInput, limit = 0) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeInsights(period, limit),
    queryFn: async () =>
      (await finance.listInsights({ period: toWirePeriod(period), limit })).insights,
  });
}


export function useRecurringPayments(includeInactive = false, asOf = "") {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeRecurringList(includeInactive, asOf),
    queryFn: async () =>
      (await finance.listRecurringPayments({ includeInactive, asOf })).payments,
  });
}

export interface CadenceInput {
  interval: number;
  unit: RecurrenceUnit;
  dayOfMonth?: number;
  dayOfWeek?: Weekday;
}

export interface CreateRecurringPaymentInput {
  name: string;
  amount: Money;
  type: TransactionType;
  categoryId: string;
  accountId: string;
  memberId?: string;
  cadence: CadenceInput;
  nextDueOn: string;
  endOn?: string;
  autoPost?: boolean;
}

function cadenceRequest(cadence: CadenceInput) {
  return {
    interval: cadence.interval,
    unit: cadence.unit,
    dayOfMonth: cadence.dayOfMonth ?? 0,
    dayOfWeek: cadence.dayOfWeek ?? Weekday.UNSPECIFIED,
  };
}

export function useCreateRecurringPayment() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateRecurringPaymentInput) =>
      (
        await finance.createRecurringPayment({
          name: input.name,
          amount: toWire(input.amount),
          type: input.type,
          categoryId: input.categoryId,
          accountId: input.accountId,
          memberId: input.memberId ?? "",
          cadence: cadenceRequest(input.cadence),
          nextDueOn: input.nextDueOn,
          endOn: input.endOn ?? "",
          autoPost: input.autoPost ?? false,
        })
      ).payment ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeRecurring }),
  });
}

export interface UpdateRecurringPaymentInput {
  recurringId: string;
  name?: string;
  categoryId?: string;
  accountId?: string;
  memberId?: string;
  nextDueOn?: string;
  endOn?: string;
  autoPost?: boolean;
  active?: boolean;
  amount?: Money;
  cadence?: CadenceInput;
}

export function useUpdateRecurringPayment() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ amount, cadence, ...rest }: UpdateRecurringPaymentInput) =>
      (
        await finance.updateRecurringPayment({
          ...rest,
          amount: amount ? toWire(amount) : undefined,
          cadence: cadence ? cadenceRequest(cadence) : undefined,
        })
      ).payment ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeRecurring }),
  });
}

export function useDeleteRecurringPayment() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (recurringId: string) => finance.deleteRecurringPayment({ recurringId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeRecurring }),
  });
}

export function usePostRecurringOccurrence() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { recurringId: string; dueOn: string; amountOverride?: Money }) =>
      finance.postRecurringOccurrence({
        recurringId: input.recurringId,
        dueOn: input.dueOn,
        amountOverride: input.amountOverride ? toWire(input.amountOverride) : undefined,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.finance }),
  });
}

export function useSkipRecurringOccurrence() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { recurringId: string; dueOn: string }) =>
      finance.skipRecurringOccurrence(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeRecurring }),
  });
}


export function useReminders(includeDisabled = false) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeRemindersList(includeDisabled),
    queryFn: async () => (await finance.listReminders({ includeDisabled })).reminders,
  });
}

export interface UpsertReminderInput {
  reminderId?: string;
  kind: ReminderKind;
  title: string;
  dueAt?: Date;
  repeat?: CadenceInput;
  enabled?: boolean;
}

export function useUpsertReminder() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpsertReminderInput) =>
      (
        await finance.upsertReminder({
          reminderId: input.reminderId ?? "",
          kind: input.kind,
          title: input.title,
          dueAt: input.dueAt ? timestampFrom(input.dueAt) : undefined,
          repeat: input.repeat ? cadenceRequest(input.repeat) : undefined,
          enabled: input.enabled ?? true,
        })
      ).reminder ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeReminders }),
  });
}

export function useDeleteReminder() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (reminderId: string) => finance.deleteReminder({ reminderId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeReminders }),
  });
}

function timestampFrom(date: Date) {
  const ms = date.getTime();
  return { seconds: BigInt(Math.floor(ms / 1000)), nanos: (ms % 1000) * 1_000_000 };
}


export function useWidgets() {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeWidgetsList(),
    queryFn: async () => (await finance.listWidgets({})).widgets,
  });
}

export interface AddWidgetInput {
  type: WidgetType;
  size: WidgetSize;
  scope?: ScopeInput;
  targetRef?: string;
  targetAccountIds?: string[];
}

export function useAddWidget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: AddWidgetInput) =>
      (
        await finance.addWidget({
          type: input.type,
          size: input.size,
          scope: toWireScope(input.scope ?? familyScope),
          targetRef: input.targetRef ?? "",
          targetAccountIds: input.targetAccountIds ?? [],
        })
      ).widget ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeWidgets }),
  });
}

export interface UpdateWidgetInput {
  widgetId: string;
  size?: WidgetSize;
  scope?: ScopeInput;
  targetRef?: string;
  targetAccountIds?: string[];
}

export function useUpdateWidget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ scope, ...rest }: UpdateWidgetInput) =>
      (await finance.updateWidget({ ...rest, scope: scope ? toWireScope(scope) : undefined }))
        .widget ?? null,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeWidgets }),
  });
}

export function useRemoveWidget() {
  const { finance } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (widgetId: string) => finance.removeWidget({ widgetId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.financeWidgets }),
  });
}

export function useWidgetData(widgetIds: readonly string[] = []) {
  const { finance } = useClients();
  return useQuery({
    queryKey: queryKeys.financeWidgetData(widgetIds),
    queryFn: async () => (await finance.getWidgetData({ widgetIds: [...widgetIds] })).payloads,
  });
}


export {
  AccountKind,
  AccountVisibility,
  FamilyMemberRole,
  InvitationStatus,
  BudgetPeriod,
  BudgetTargetFilter,
  BudgetTargetKind,
  InsightKind,
  MemberRole,
  MemberStatus,
  RecurrenceUnit,
  ReminderKind,
  ScopeKind,
  SeriesStacking,
  TransactionKind,
  TransactionType,
  Weekday,
  WidgetSize,
  WidgetType,
};

export type { Family, FamilyMember, Invitation };

export type {
  Account,
  Budget,
  BudgetStatus,
  Cadence,
  Category,
  CategoryGroup,
  CategorySlice,
  DaySection,
  DonutSlice,
  GroupMemberSplit,
  GroupNode,
  GroupRow,
  HiddenPrivateSummary,
  HouseholdFinanceSettings,
  Insight,
  Member,
  MemberAmount,
  MemberChip,
  MemberSpending,
  QuickTemplate,
  RecurringPayment,
  RecurringPaymentStatus,
  Reminder,
  SeriesBucket,
  SeriesSegment,
  Transaction,
  WidgetInstance,
  WidgetPayload,
} from "@fm/sdk/finance/v1/finance_pb";
