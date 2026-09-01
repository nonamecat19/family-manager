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

/**
 * Commonplace's data layer. The rules that hold across this file:
 *
 *  - a query unwraps the response field it is named after, unless the screen needs the rest
 *    of the response too (search's timing footer, "shared with me" returning two lists);
 *  - a mutation invalidates the *notes domain* key. One edit moves the list row, the
 *    notebook's note_count, the note detail and the activity rail, and invalidating those by
 *    hand is how two panes end up disagreeing about the same note;
 *  - permission is never re-derived here. `note.canEdit` is the server's answer and the app
 *    reads it — the proto says so, and the copy in JavaScript is the one that drifts.
 *
 * Two paths are optimistic, because the design gives them no confirmation step: the star on
 * a list row, and the todo checkbox in the editor. Everything else waits for the server.
 */

export {
  blocksToPlainText,
  countTasks,
  type BlockLike,
  type TaskCounts,
} from "./noteBlocks.ts";

/**
 * SearchFacet.ALL as its wire number, typed as the enum: the generated enum is a TS `enum`
 * and this module deliberately keeps its SDK imports type-only (see noteBlocks.ts).
 */
const FACET_ALL: SearchFacet = 1;

/* ------------------------------------------------------------------ notebooks */

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
  /** Empty moves the notebook to the top level. */
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

/**
 * Deleting a notebook is refused server-side while it still holds notes (see the proto), so
 * this is not optimistic: the failure is an ordinary one the sheet has to show.
 */
export function useDeleteNotebook() {
  const { notes } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (notebookId: string) => notes.deleteNotebook({ notebookId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

/* ---------------------------------------------------------------------- notes */

export interface NoteListFilters {
  /** Empty means "every notebook"; set it to scope the list to one. */
  notebookId?: string;
  starredOnly?: boolean;
  includeArchived?: boolean;
  /** Notes the caller can see but does not own — the sidebar's "Shared with me" shape. */
  sharedOnly?: boolean;
  /**
   * The Archive rail row: archived notes and nothing else. It implies `includeArchived`, so
   * the two never have to be sent together. Ask for it rather than fetching `includeArchived`
   * and dropping the live rows here — the page is cut server-side, so with a `pageSize` set
   * the client-side filter throws away a page that never held the archive in the first place.
   */
  archivedOnly?: boolean;
  sort?: NoteSort;
  pageSize?: number;
}

/** Every notes list cache entry, for the optimistic paths below. */
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
  /** Empty leaves the note outside every notebook — where quick capture puts it. */
  notebookId?: string;
  title: string;
  blocks: Block[];
  /**
   * The id the app minted while offline. Passing it makes the create idempotent for this
   * user, which is what lets the capture queue retry a request whose answer it never saw.
   */
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
  /** The whole block array. UpdateNote writes the document, not a diff — see the proto. */
  blocks: Block[];
  /**
   * The version last read. The write is refused with ABORTED when it no longer matches, which
   * is how two people editing one shared note find out. Send 0 only to force, after showing
   * the conflict.
   */
  expectedVersion?: bigint;
  /**
   * Set by the editor's todo checkbox. A checkbox cannot wait for a round trip, so this path
   * paints the new blocks into the cache immediately and rolls them back if the write fails.
   * A body edit leaves it unset: the editor already holds that text on screen, and painting
   * the cache mid-typing only fights the input.
   */
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
      // Task counts are recomputed locally so the list row's "6/9" moves with the checkbox
      // instead of waiting for the server's own count to come back.
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
    // Both notebooks' note_count moved, so the domain-wide invalidation is the point here.
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

/**
 * The star on a list row. Optimistic for the same reason the checkbox is: the control has no
 * confirmation step in the design, and a star that lags the tap reads as a dropped tap.
 */
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
    // The starred-only list is a different query: only a domain invalidation adds or removes
    // the row there.
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.notes }),
  });
}

/**
 * The Archive action, in the note header's overflow menu and the list row's. Not optimistic:
 * archiving removes the row from the list it was tapped in, and a row that vanishes before the
 * server has agreed is a row that has to be put back if the write fails. The domain-wide
 * invalidation repaints the list, the archive and the notebook counts together.
 *
 * `archived` is the value to set, not a toggle — pass `!note.archived` to flip it — so a
 * retried request lands on the state the caller asked for.
 */
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

/**
 * NOT USED IN V1, and no app should start using it without the storage change described below.
 * It is kept because `UploadNoteImage` is in the contract and a client wrapper for an unused
 * rpc costs nothing; a docstring describing a flow that does not ship costs a wrong feature.
 *
 * Image blocks are deferred: `libs/go/storage` puts an anonymous-read policy on every bucket it
 * creates, so a photo inside a note that was never shared would be fetchable by anyone holding
 * its URL, and un-sharing the note would not take it back. Notes are private by default, so no
 * deployment of the service is given a `NOTES_STORAGE_ENDPOINT` — neither compose file passes
 * one — and without it the service provisions no bucket and the rpc answers `unimplemented`.
 * (Set that variable locally and the bucket, and the anonymous-read policy on it, come back:
 * the guarantee is a deployment fact, not something the code can enforce.) `apps/notes` offers
 * no picker, and an image block from an older client is rendered as a placeholder and re-saved
 * untouched.
 *
 * If it ever ships: raw bytes over Connect/JSON, one round trip — the same tradeoff the recipes
 * image upload makes — and the caller stores the returned URL on the block and saves the note.
 */
export function useUploadNoteImage() {
  const { notes } = useClients();
  return useMutation({
    mutationFn: async (input: { noteId: string; image: Uint8Array; contentType: string }) => {
      const res = await notes.uploadNoteImage(input);
      return res.imageUrl;
    },
  });
}

/* --------------------------------------------------------------------- search */

/**
 * The ⌘K palette. Search is server-side on purpose (the proto says why): the app must not
 * filter a full note list, which stops working at the first notebook nobody opened.
 *
 * Debounce-friendly by construction — an empty query is disabled rather than sent, so a
 * caller can pass the raw input value while its debounced copy catches up and no request
 * goes out for "". The whole response comes back: the footer prints elapsedMs and
 * searchedNotes ("Searched 128 notes in 31ms").
 */
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

/* -------------------------------------------------------------------- sharing */

/** A share always hangs off exactly one of a note or a notebook. */
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
  /** Required when subject is MEMBER, ignored when it is FAMILY. */
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
    // Sharing changes who can see the note, so the recipient's lists change too — and the
    // note's own row grows the "people" glyph. Domain-wide.
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

/** Revokes one share row. The target must match where the share was granted. */
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

/**
 * The sidebar's "Shared with me". The whole response comes back because it carries two
 * lists — notes and notebooks — and the section draws both.
 */
export function useSharedWithMe(pageSize = 0) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.sharedWithMe(),
    queryFn: () => notes.listSharedWithMe({ pageSize }),
  });
}

/* ------------------------------------------------------------------- comments */

/**
 * Named `useNoteComments`, not `useComments`: @fm/api already exports a `useComments` for
 * recipes, and the barrel would make the two ambiguous.
 */
export function useNoteComments(noteId: string, includeResolved = false) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.noteComments(noteId, includeResolved),
    queryFn: async () => (await notes.listComments({ noteId, includeResolved })).comments,
    enabled: noteId !== "",
  });
}

/** Likewise prefixed to stay clear of the recipes `useAddComment`. */
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

/* ------------------------------------------------------------------- activity */

/** The note's Activity rail. The server sends kinds, not prose; the app writes the sentence. */
export function useActivity(noteId: string, limit = 0) {
  const { notes } = useClients();
  return useQuery({
    queryKey: queryKeys.noteActivity(noteId),
    queryFn: async () => (await notes.listActivity({ noteId, limit })).activity,
    enabled: noteId !== "",
  });
}

/* ----------------------------------------------------------------- optimistic */

/**
 * The cache entries an optimistic note edit touched, so onError can put them back exactly as
 * they were. Untyped values on purpose: a snapshot is only ever handed back to setQueryData.
 */
type PreviousNotes = [readonly unknown[], unknown][];

/** Stops in-flight reads that would land on top of the optimistic paint, then snapshots. */
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

/**
 * Applies `patch` to one note wherever it is cached — the detail entry and every list that
 * happens to hold the row. Lists that do not contain the note are returned by reference, so
 * React Query does not repaint a pane the edit never touched.
 */
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
