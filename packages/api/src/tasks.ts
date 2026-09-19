import { create } from "@bufbuild/protobuf";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import {
  Task,
  TaskStatus,
  Priority,
  TaskFilter,
  AssigneeList,
  Birthday,
  FamilySettings,
  GoogleConnection,
  GoogleCalendar,
  SnoozeOption,
  ListTasksRequestSchema,
  GetTaskRequestSchema,
  CreateTaskRequestSchema,
  UpdateTaskRequestSchema,
  CompleteTaskRequestSchema,
  ReopenTaskRequestSchema,
  DeleteTaskRequestSchema,
  ListBirthdaysRequestSchema,
  CreateBirthdayRequestSchema,
  UpdateBirthdayRequestSchema,
  DeleteBirthdayRequestSchema,
  GetFamilySettingsRequestSchema,
  SetFamilySettingsRequestSchema,
  ConnectGoogleRequestSchema,
  DisconnectGoogleRequestSchema,
  ListGoogleCalendarsRequestSchema,
  SetGoogleCalendarRequestSchema,
  ListMyDueRemindersRequestSchema,
  AckReminderRequestSchema,
  SnoozeReminderRequestSchema,
  GetDigestRequestSchema,
  MarkDigestSentRequestSchema,
  AssigneeListSchema,
  GetGoogleConnectionRequestSchema,
} from "@fm/sdk/tasks/v1/tasks_pb";
// TasksService is used via useClients()

import { useClients } from "./hooks.ts";
import { queryKeys } from "./queryKeys.ts";
import type { TaskListFilters, BirthdayListFilters } from "./queryKeys.ts";

export type { Task, TaskStatus, Priority, TaskFilter, AssigneeList, Birthday, FamilySettings, GoogleConnection, GoogleCalendar, SnoozeOption };
export type { TaskListFilters, BirthdayListFilters } from "./queryKeys.ts";

export function useTasks(filters: TaskListFilters = {}) {
  const { tasks } = useClients();
  const key = queryKeys.tasksList(filters);

  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const res = await tasks.listTasks(create(ListTasksRequestSchema, {
        filter: filters.filter === "all" ? TaskFilter.ALL : filters.filter === "done" ? TaskFilter.DONE : TaskFilter.MINE,
      }));
      return res.tasks;
    },
  });
}

export function useTask(id: string) {
  const { tasks } = useClients();
  const key = queryKeys.task(id);

  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const res = await tasks.getTask(create(GetTaskRequestSchema, { taskId: id }));
      return res.task;
    },
    enabled: id !== "",
  });
}

export function useCreateTask() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: { title: string; notes?: string; priority?: Priority; dueOn?: string; dueTime?: string; assigneeUserIds?: string[] }) => {
      const res = await tasks.createTask(create(CreateTaskRequestSchema, {
        title: input.title,
        notes: input.notes ?? "",
        priority: input.priority ?? Priority.MEDIUM,
        dueOn: input.dueOn ?? "",
        dueTime: input.dueTime ?? "",
        assigneeUserIds: input.assigneeUserIds ?? [],
      }));
      return res.task;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks });
    },
  });
}

export function useUpdateTask() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: { id: string; title?: string; notes?: string; priority?: Priority; dueOn?: string; dueTime?: string; assigneeUserIds?: string[] }) => {
      const res = await tasks.updateTask(create(UpdateTaskRequestSchema, {
        taskId: input.id,
        title: input.title,
        notes: input.notes,
        priority: input.priority,
        dueOn: input.dueOn,
        dueTime: input.dueTime,
        assignees: create(AssigneeListSchema, { userIds: input.assigneeUserIds ?? [] }),
      }));
      return res.task;
    },
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: queryKeys.task(vars.id) });
      qc.invalidateQueries({ queryKey: queryKeys.tasks });
    },
  });
}

export function useCompleteTask() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const res = await tasks.completeTask(create(CompleteTaskRequestSchema, { taskId: id }));
      return res.task;
    },
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: queryKeys.task(id) });
      qc.invalidateQueries({ queryKey: queryKeys.tasks });
    },
  });
}

export function useReopenTask() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const res = await tasks.reopenTask(create(ReopenTaskRequestSchema, { taskId: id }));
      return res.task;
    },
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: queryKeys.task(id) });
      qc.invalidateQueries({ queryKey: queryKeys.tasks });
    },
  });
}

export function useDeleteTask() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await tasks.deleteTask(create(DeleteTaskRequestSchema, { taskId: id }));
    },
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: queryKeys.task(id) });
      qc.invalidateQueries({ queryKey: queryKeys.tasks });
    },
  });
}

export function useBirthdays(filters: BirthdayListFilters = {}) {
  const { tasks } = useClients();
  const key = queryKeys.birthdaysList(filters);

  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const res = await tasks.listBirthdays(create(ListBirthdaysRequestSchema, {}));
      return res.birthdays;
    },
  });
}

export function useCreateBirthday() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name: string; day: number; month: number; year?: number; remindDaysBefore?: number }) => {
      const res = await tasks.createBirthday(create(CreateBirthdayRequestSchema, {
        name: input.name,
        day: input.day,
        month: input.month,
        year: input.year ?? 0,
        remindDaysBefore: input.remindDaysBefore ?? 0,
      }));
      return res.birthday;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks });
    },
  });
}

export function useUpdateBirthday() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: { id: string; name?: string; day?: number; month?: number; year?: number; remindDaysBefore?: number }) => {
      const res = await tasks.updateBirthday(create(UpdateBirthdayRequestSchema, {
        birthdayId: input.id,
        name: input.name,
        day: input.day,
        month: input.month,
        year: input.year,
        remindDaysBefore: input.remindDaysBefore,
      }));
      return res.birthday;
    },
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: queryKeys.birthday(vars.id) });
      qc.invalidateQueries({ queryKey: queryKeys.tasks });
    },
  });
}

export function useDeleteBirthday() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await tasks.deleteBirthday(create(DeleteBirthdayRequestSchema, { birthdayId: id }));
    },
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: queryKeys.birthday(id) });
      qc.invalidateQueries({ queryKey: queryKeys.tasks });
    },
  });
}

export function useFamilySettings() {
  const { tasks } = useClients();
  const key = queryKeys.taskSettings({});

  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const res = await tasks.getFamilySettings(create(GetFamilySettingsRequestSchema, { deviceTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone }));
      return res.settings;
    },
  });
}

export function useSetFamilySettings() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (timezone: string) => {
      const res = await tasks.setFamilySettings(create(SetFamilySettingsRequestSchema, { timezone }));
      return res.settings;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.taskSettings({}) });
    },
  });
}

export function useGoogleConnection() {
  const { tasks } = useClients();
  const key = queryKeys.taskSettings({ includeGoogle: true });

  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const res = await tasks.getGoogleConnection(create(GetGoogleConnectionRequestSchema, {}));
      return res.connection;
    },
  });
}

export function useGoogleCalendars() {
  const { tasks } = useClients();
  const key = queryKeys.taskSettings({ includeGoogle: true });

  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const res = await tasks.listGoogleCalendars(create(ListGoogleCalendarsRequestSchema, {}));
      return res.calendars;
    },
  });
}

export function useConnectGoogle() {
  const { tasks } = useClients();

  return useMutation({
    mutationFn: async (input: { code: string; codeVerifier: string; redirectUri: string }) => {
      const res = await tasks.connectGoogle(create(ConnectGoogleRequestSchema, {
        code: input.code,
        codeVerifier: input.codeVerifier,
        redirectUri: input.redirectUri,
      }));
      return res.connection;
    },
  });
}

export function useDisconnectGoogle() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      await tasks.disconnectGoogle(create(DisconnectGoogleRequestSchema, {}));
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.taskSettings({ includeGoogle: true }) });
    },
  });
}

export function useSetGoogleCalendar() {
  const { tasks } = useClients();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (calendarId: string) => {
      const res = await tasks.setGoogleCalendar(create(SetGoogleCalendarRequestSchema, { calendarId }));
      return res.connection;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.taskSettings({ includeGoogle: true }) });
    },
  });
}

export function useMyDueReminders() {
  const { tasks } = useClients();
  const key = queryKeys.taskSettings({ includeGoogle: true });

  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const res = await tasks.listMyDueReminders(create(ListMyDueRemindersRequestSchema, {}));
      return res.reminders;
    },
  });
}

export function useAckReminder() {
  const { tasks } = useClients();

  return useMutation({
    mutationFn: async (reminderId: string) => {
      await tasks.ackReminder(create(AckReminderRequestSchema, { reminderId }));
    },
  });
}

export function useSnoozeReminder() {
  const { tasks } = useClients();

  return useMutation({
    mutationFn: async (input: { reminderId: string; option: SnoozeOption }) => {
      const res = await tasks.snoozeReminder(create(SnoozeReminderRequestSchema, {
        reminderId: input.reminderId,
        option: input.option,
      }));
      return res.remindAt;
    },
  });
}

export function useDigest() {
  const { tasks } = useClients();
  const key = queryKeys.taskSettings({ includeGoogle: true });

  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const res = await tasks.getDigest(create(GetDigestRequestSchema, {}));
      return res;
    },
  });
}

export function useMarkDigestSent() {
  const { tasks } = useClients();

  return useMutation({
    mutationFn: async (date: string) => {
      await tasks.markDigestSent(create(MarkDigestSentRequestSchema, { date }));
    },
  });
}