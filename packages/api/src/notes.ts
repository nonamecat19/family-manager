import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  Block,
  Note,
  NoteSort,
  SearchFacet,
  SharePermission,
  ShareSubject,
} from "@fm/sdk/notes/v1/notes_pb";

import { blocksToPlainText, countTasks } from "./noteBlocks.ts";
import { useClients } from "./provider.tsx";
import { queryKeys } from "./queryKeys.ts";


export {
  blocksToPlainText,
  countTasks,
  type BlockLike,
  type TaskCounts,
} from "./noteBlocks.ts";

const FACET_ALL: SearchFacet = 1;


export function useNotebooks(includeArchived = false) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.notebooks(includeArchived),
    queryFn: async () => (await notes.listNotebooks({ includeArchived })).notebooks,
  });
}

export function useCreateNotebook() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { name: string; parentId?: string }) => {
      const res = await notes.createNotebook({ name: input.name, parentId: input.parentId ?? "" });
      return res.notebook;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export interface UpdateNotebookInput {
  notebookId: string;
  name: string;
  parentId?: string;
  archived?: boolean;
}

export function useUpdateNotebook() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateNotebookInput) => {
      const res = await notes.updateNotebook({
        notebookId: input.notebookId,
        name: input.name,
        parentId: input.parentId ?? "",
        archived: input.archived ?? false,
      });
      return res.notebook;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useDeleteNotebook() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (notebookId: string) => notes.deleteNotebook({ notebookId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}


export interface NoteListFilters {
  notebookId?: string;
  starredOnly?: boolean;
  includeArchived?: boolean;
  sharedOnly?: boolean;
  archivedOnly?: boolean;
  sort?: NoteSort;
  pageSize?: number;
}

const NOTE_LISTS = { queryKey: [...queryKeys.notes, "list"] } as const;

export function useNotes(filters: NoteListFilters = {}, opts: { enabled?: boolean } = {}) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.notesList(filters),
    queryFn: async () =>
      (
        await notes.listNotes({
          notebookId: filters.notebookId ?? "",
          starredOnly: filters.starredOnly ?? false,
          includeArchived: filters.includeArchived ?? false,
          archivedOnly: filters.archivedOnly ?? false,
          sharedOnly: filters.sharedOnly ?? false,
          sort: filters.sort,
          pageSize: filters.pageSize ?? 0,
        })
      ).notes,
    enabled: opts.enabled ?? true,
  });
}

export function useNote(id: string) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.note(id),
    queryFn: async () => {
      const res = await notes.getNote({ noteId: id });
      return res.note ?? null;
    },
    enabled: id !== "",
  });
}

export interface CreateNoteInput {
  notebookId?: string;
  title: string;
  blocks: Block[];
  clientId?: string;
}

export function useCreateNote() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateNoteInput) => {
      const res = await notes.createNote({
        notebookId: input.notebookId ?? "",
        title: input.title,
        blocks: input.blocks,
        clientId: input.clientId ?? "",
      });
      return res.note;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export interface UpdateNoteInput {
  noteId: string;
  title: string;
  blocks: Block[];
  expectedVersion?: bigint;
  optimistic?: boolean;
}

export function useUpdateNote() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateNoteInput) => {
      const res = await notes.updateNote({
        noteId: input.noteId,
        title: input.title,
        blocks: input.blocks,
        expectedVersion: input.expectedVersion ?? 0n,
      });
      return res.note;
    },
    onMutate: async (input: UpdateNoteInput) => {
      if (!input.optimistic) return { previous: [] as PreviousNotes };
      const previous = await snapshotNote(qc, input.noteId);
      const counts = countTasks(input.blocks);
      patchNote(qc, input.noteId, (note) => ({
        ...note,
        title: input.title,
        blocks: input.blocks,
        preview: blocksToPlainText(input.blocks).slice(0, 140),
        taskTotal: counts.total,
        taskDone: counts.done,
      }));
      return { previous };
    },
    onError: (_error, _input, context) => restoreNotes(qc, context?.previous),
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useMoveNote() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { noteId: string; notebookId: string }) => {
      const res = await notes.moveNote(input);
      return res.note;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useToggleStar() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { noteId: string; starred: boolean }) => {
      const res = await notes.toggleStar(input);
      return res.note;
    },
    onMutate: async (input: { noteId: string; starred: boolean }) => {
      const previous = await snapshotNote(qc, input.noteId);
      patchNote(qc, input.noteId, (note) => ({ ...note, starred: input.starred }));
      return { previous };
    },
    onError: (_error, _input, context) => restoreNotes(qc, context?.previous),
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useArchiveNote() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { noteId: string; archived: boolean }) => {
      const res = await notes.archiveNote(input);
      return res.note;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useDeleteNote() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (noteId: string) => notes.deleteNote({ noteId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useUploadNoteImage() {
  const { notes } = useClients();
  return useMutation({
    mutationFn: async (input: { noteId: string; image: Uint8Array; contentType: string }) => {
      const res = await notes.uploadNoteImage(input);
      return res.imageUrl;
    },
  });
}


export function useNoteSearch(
  query: string,
  facet: SearchFacet = FACET_ALL,
  opts: { limit?: number; enabled?: boolean } = {},
) {
  const { notes } = useClients();
  const trimmed = query.trim();
  return useQuery({
    queryKey: queryKeys.noteSearch(trimmed, facet),
    queryFn: () => notes.search({ query: trimmed, facet, limit: opts.limit ?? 0 }),
    enabled: trimmed !== "" && (opts.enabled ?? true),
  });
}


export interface ShareTarget {
  noteId?: string;
  notebookId?: string;
}

export function useShares(target: ShareTarget) {
  const { notes } = useClients();
  const noteId = target.noteId ?? "";
  const notebookId = target.notebookId ?? "";
  return useQuery({
    queryKey: notebookId !== ""
      ? queryKeys.noteShares(notebookId, "notebook")
      : queryKeys.noteShares(noteId, "note"),
    queryFn: async () => (await notes.listShares({ noteId, notebookId })).shares,
    enabled: noteId !== "" || notebookId !== "",
  });
}

export interface ShareInput {
  subject: ShareSubject;
  memberUserId?: string;
  permission: SharePermission;
}

export function useShareNote() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ShareInput & { noteId: string }) => {
      const res = await notes.shareNote({
        noteId: input.noteId,
        subject: input.subject,
        memberUserId: input.memberUserId ?? "",
        permission: input.permission,
      });
      return res.share;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useShareNotebook() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ShareInput & { notebookId: string }) => {
      const res = await notes.shareNotebook({
        notebookId: input.notebookId,
        subject: input.subject,
        memberUserId: input.memberUserId ?? "",
        permission: input.permission,
      });
      return res.share;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useUnshare() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ShareTarget & { shareId: string }) =>
      notes.unshare({
        shareId: input.shareId,
        noteId: input.noteId ?? "",
        notebookId: input.notebookId ?? "",
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useSharedWithMe(pageSize = 0) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.sharedWithMe(),
    queryFn: () => notes.listSharedWithMe({ pageSize }),
  });
}


export function useNoteComments(noteId: string, includeResolved = false) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.noteComments(noteId, includeResolved),
    queryFn: async () => (await notes.listComments({ noteId, includeResolved })).comments,
    enabled: noteId !== "",
  });
}

export function useAddNoteComment() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { noteId: string; body: string }) => {
      const res = await notes.addComment(input);
      return res.comment;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

export function useResolveComment() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: { commentId: string; resolved: boolean }) => {
      const res = await notes.resolveComment(input);
      return res.comment;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}


export function useActivity(noteId: string, limit = 0) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.noteActivity(noteId),
    queryFn: async () => (await notes.listActivity({ noteId, limit })).activity,
    enabled: noteId !== "",
  });
}


type PreviousNotes = [readonly unknown[], unknown][];

async function snapshotNote(
  qc: ReturnType<typeof useQueryClient>,
  noteId: string,
): Promise<PreviousNotes> {
  await qc.cancelQueries({ queryKey: queryKeys.note(noteId) });
  await qc.cancelQueries(NOTE_LISTS);
  return [
    [queryKeys.note(noteId), qc.getQueryData(queryKeys.note(noteId))],
    ...qc.getQueriesData(NOTE_LISTS),
  ];
}

function restoreNotes(qc: ReturnType<typeof useQueryClient>, previous: PreviousNotes | undefined) {
  for (const [key, data] of previous ?? []) qc.setQueryData(key, data);
}

function patchNote(
  qc: ReturnType<typeof useQueryClient>,
  noteId: string,
  patch: (note: Note) => Note,
) {
  qc.setQueryData<Note | null>(queryKeys.note(noteId), (note) => (note ? patch(note) : note));
  qc.setQueriesData<readonly Note[]>(NOTE_LISTS, (list) => {
    if (!list) return list;
    let found = false;
    const next = list.map((note) => {
      if (note.id !== noteId) return note;
      found = true;
      return patch(note);
    });
    return found ? next : list;
  });
}
