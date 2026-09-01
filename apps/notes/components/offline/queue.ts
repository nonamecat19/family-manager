import { Code, ConnectError } from "@connectrpc/connect";
import { useCreateNote } from "@fm/api";
import { useAuth } from "@fm/auth";
import NetInfo from "@react-native-community/netinfo";
import { File, Paths } from "expo-file-system";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { ownerFileKey, userIdFromToken } from "./identity.ts";
import {
  QueueOwnerError,
  createQueueStore,
  newClientId,
  type QueuedNote,
  type QueueStorage,
} from "./store.ts";

/**
 * The offline capture queue.
 *
 * Quick capture on a phone has to work with no network — that is the whole point of the sheet
 * — so a captured note is written to a JSON file FIRST and turned into a CreateNote later.
 *
 * Decisions worth keeping:
 *
 *  - THE FILE IS ONE USER'S. Its name carries the signed-in user's id and its contents name the
 *    owner again, and store.ts refuses to adopt a file stamped with anybody else. The single
 *    `capture-queue.json` this replaced was the whole product promise inverted on a family's
 *    shared tablet: a note user A wrote offline and never flushed was created in user B's
 *    account the moment B signed in, because the queue had no idea whose notes it held.
 *  - SIGNING OUT FORGETS THE QUEUE, IT DOES NOT DELETE IT. `resetCaptureQueue()` drops the
 *    in-memory list so nothing of A's is on screen or in reach for B; A's file stays on the
 *    disk, and A gets their unsent writing back the next time they sign in on this device.
 *    Settings tries a flush first, so in the ordinary case there is nothing left to keep.
 *  - The file lives in expo-file-system's document directory, not SecureStore. SecureStore is
 *    the keychain, it is for the auth tokens, and it has a small value limit; a note body is
 *    ordinary user data with no business being in there.
 *  - Every queued note carries a client-minted `client_id`, minted ONCE in `capture` and
 *    persisted with the draft, and it is sent as CreateNoteRequest.client_id on every attempt.
 *    The proto makes that create idempotent per user, which is what makes a retry safe after a
 *    request whose answer we never saw: the second call returns the note the first one made
 *    instead of a duplicate. Minting it at flush time instead would turn "flush on reconnect"
 *    into a duplicate generator, so the id is never regenerated after capture.
 *
 * The ordering and ownership rules live in store.ts, where they can be tested without a native
 * module; reading the owner out of the access token lives in identity.ts, for the same reason.
 */

export type { QueuedNote, QueueRejection } from "./store.ts";
export { newClientId } from "./store.ts";

/** How long the queue waits after a failed flush before it tries the same notes again. */
const RETRY_COOLDOWN_MS = 15_000;

function handle(ownerId: string): File | null {
  try {
    return new File(Paths.document, `capture-queue-${ownerFileKey(ownerId)}.json`);
  } catch {
    // Web and any runtime without a document directory: the queue still works for the
    // session, it just does not outlive it. Better than a capture sheet that throws.
    return null;
  }
}

const fileStorage: QueueStorage = {
  read: async (ownerId) => {
    const file = handle(ownerId);
    if (!file || !file.exists) return null;
    return await file.text();
  },
  write: async (ownerId, contents) => {
    const file = handle(ownerId);
    if (!file) return;
    if (!file.exists) file.create({ intermediates: true, overwrite: true });
    file.write(contents);
  },
};

const store = createQueueStore(fileStorage);

/**
 * The flush guard. It lives at module scope because the thing it protects does: `store` above
 * is one queue for the whole app, and three components mount `useCaptureQueue()` on the same
 * route — the list, the capture sheet and Settings. As per-hook refs these flags gave each
 * mount its own "not busy", so two of them could walk the same pending list at the same time
 * and send every note twice. (`client_id` makes the duplicate harmless on the server; it does
 * not make it free.)
 */
let flushing = false;
/**
 * When the queue may next try again. A failed flush must not be retried on the very next
 * render — that is a tight loop against a server that just said no. One cooldown, then the
 * next reconnect or screen visit tries.
 */
let retryAfter = 0;

/** Mounted hooks that want to show a spinner while the one shared flush is running. */
const syncListeners = new Set<(next: boolean) => void>();

function setFlushing(next: boolean) {
  flushing = next;
  for (const listener of syncListeners) listener(next);
}

export const queueSnapshot = store.snapshot;
export const queueOwner = store.owner;
export const subscribeToQueue = store.subscribe;
export const hydrateQueue = store.hydrate;
export const enqueueNote = store.enqueue;
export const dequeueNote = store.dequeue;

/**
 * Forgets everything the queue holds in memory and who it belonged to. Called on every
 * authenticated → anonymous transition (app/_layout.tsx), which covers the Settings button, a
 * refresh the server rejected, and any other way a session ends.
 *
 * The owner's file is left on the disk on purpose — see the note at the top of this file.
 */
export async function resetCaptureQueue(): Promise<void> {
  retryAfter = 0;
  await store.reset();
}

export interface CaptureQueue {
  /** Everything the queue holds, refused notes included — what the list draws as pending rows. */
  queued: QueuedNote[];
  /** The notes a flush will actually try to send. */
  pending: QueuedNote[];
  /** Notes the server refused for a reason a retry cannot fix. They wait for the user. */
  rejected: QueuedNote[];
  /** NetInfo's answer. The capture sheet's "Offline — saves locally" pill reads this. */
  online: boolean;
  syncing: boolean;
  /** The signed-in user the queue is scoped to; null until the token has been read. */
  ownerId: string | null;
  capture: (note: Omit<QueuedNote, "clientId" | "capturedAt" | "rejection">) => Promise<void>;
  /** Sends everything the queue holds. Called on reconnect, and from Settings. */
  flush: () => Promise<void>;
  /** Moves a refused note out of the notebook that refused it and tries again. */
  refile: (clientId: string, notebookId?: string) => Promise<void>;
}

/**
 * The queue as React sees it: the pending notes, whether the device is online, and a flush
 * that fires by itself the moment connectivity comes back.
 */
export function useCaptureQueue(): CaptureQueue {
  const { status, getAccessToken } = useAuth();
  const [queued, setQueued] = useState<QueuedNote[]>(queueSnapshot);
  const [online, setOnline] = useState(true);
  const [syncing, setSyncing] = useState(flushing);
  const [ownerId, setOwnerId] = useState<string | null>(queueOwner);
  const createNote = useCreateNote();
  // useMutation hands back a NEW object every render, so a `flush` built over it changes
  // identity every render — and `flush` sits in the reconnect effect's dep list, which then
  // re-runs every render. The mutation is read through a ref instead, which lets `flush`
  // depend on nothing and stay the same function for the life of the mount.
  const createNoteRef = useRef(createNote);
  // Same reason, for the owner: the callbacks below must not change identity when the token
  // finally resolves.
  const ownerRef = useRef<string | null>(ownerId);

  useEffect(() => {
    createNoteRef.current = createNote;
  });

  useEffect(() => {
    ownerRef.current = ownerId;
  }, [ownerId]);

  // Whose queue this is. The access token is the only identity the app holds, so the owner is
  // read out of it; until it resolves the queue persists nothing, which is the safe direction.
  useEffect(() => {
    let cancelled = false;
    if (status !== "authenticated") {
      setOwnerId(null);
      return;
    }
    void getAccessToken().then(
      (token) => {
        if (!cancelled) setOwnerId(userIdFromToken(token));
      },
      (error) => {
        console.warn("[notes] could not read the signed-in user for the capture queue", error);
      },
    );
    return () => {
      cancelled = true;
    };
  }, [status, getAccessToken]);

  useEffect(() => subscribeToQueue(setQueued), []);

  useEffect(() => {
    if (!ownerId) {
      setQueued(queueSnapshot());
      return;
    }
    // Awaiting hydration and reading the queue are two different things; ask the store for the
    // list only once the disk has been read.
    void hydrateQueue(ownerId).then(() => setQueued(queueSnapshot()));
  }, [ownerId]);

  useEffect(() => {
    syncListeners.add(setSyncing);
    setSyncing(flushing);
    return () => {
      syncListeners.delete(setSyncing);
    };
  }, []);

  useEffect(
    () =>
      NetInfo.addEventListener((state) => {
        setOnline(state.isConnected !== false);
      }),
    [],
  );

  const flush = useCallback(async () => {
    const owner = ownerRef.current;
    if (!owner) return;
    if (flushing || Date.now() < retryAfter) return;
    // Claimed synchronously, before the first await: two mounts calling flush in the same tick
    // must not both get past the line above.
    setFlushing(true);
    try {
      // Await the one-time hydration, then ask for the queue as it stands NOW. Reading the
      // list out of hydrateQueue()'s resolved value instead is what broke this: that promise
      // is memoised, so its value was frozen at app-start content and a note captured after
      // boot was never in the work list — it sat on disk and never synced.
      await hydrateQueue(owner);
      if (queueOwner() !== owner) return;
      // Refused notes are skipped, not retried: one note nobody can file must not sit at the
      // head of the queue and hold every note behind it hostage.
      const pending = queueSnapshot().filter((item) => !item.rejection);
      if (pending.length === 0) return;
      for (const item of pending) {
        try {
          await createNoteRef.current.mutateAsync({
            notebookId: item.notebookId,
            title: item.title,
            blocks: item.blocks,
            // The id the capture minted. Sending the same one on every attempt is the whole
            // reason a retry cannot duplicate the note.
            clientId: item.clientId,
          });
        } catch (error) {
          const refusal = terminalRefusal(error);
          if (refusal === null) throw error;
          // The server has answered, and its answer will not change: keep the writing, record
          // why, and let Settings offer to refile it. Dropping it here would lose a note the
          // user typed; retrying it forever would be a queue that never empties.
          await store.reject(owner, item.clientId, { reason: refusal, at: Date.now() });
          continue;
        }
        await dequeueNote(owner, item.clientId);
      }
    } catch (error) {
      if (error instanceof QueueOwnerError) {
        // Signed out mid-flush. Nothing to report and nothing to retry.
        return;
      }
      // Left in the queue on purpose: the client_id makes the next attempt idempotent, so
      // the worst case of retrying is one wasted request, and the best case of giving up is
      // a note the user wrote and never sees again.
      console.warn("[notes] capture queue flush stopped", error);
      retryAfter = Date.now() + RETRY_COOLDOWN_MS;
    } finally {
      setFlushing(false);
    }
  }, []);

  const capture = useCallback(
    async (note: Omit<QueuedNote, "clientId" | "capturedAt" | "rejection">) => {
      const owner = ownerRef.current;
      // No owner, no capture. Writing an unattributed file is how the previous queue handed
      // one person's note to the next person to sign in; the sheet keeps the user's text and
      // says nothing was saved, which is the truth.
      if (!owner) throw new QueueOwnerError("the signed-in user", null);
      await hydrateQueue(owner);
      await enqueueNote(owner, { ...note, clientId: newClientId(), capturedAt: Date.now() });
      if (online) await flush();
    },
    [flush, online],
  );

  const refile = useCallback(
    async (clientId: string, notebookId = "") => {
      const owner = ownerRef.current;
      if (!owner) return;
      await store.refile(owner, clientId, notebookId);
      retryAfter = 0;
      await flush();
    },
    [flush],
  );

  useEffect(() => {
    if (online && ownerId && queued.some((item) => !item.rejection)) void flush();
  }, [online, ownerId, queued, flush]);

  const pending = useMemo(() => queued.filter((item) => !item.rejection), [queued]);
  const rejected = useMemo(() => queued.filter((item) => item.rejection), [queued]);

  return { queued, pending, rejected, online, syncing, ownerId, capture, flush, refile };
}

/**
 * Whether a failed create is an answer rather than an accident.
 *
 * PermissionDenied and NotFound are what capturing into a notebook shared VIEW-only produces —
 * ListNotebooks returns those notebooks and carries no can_edit, so the sheet cannot know until
 * the server says so. InvalidArgument and FailedPrecondition are the same kind of answer. Retry
 * cannot change any of them.
 *
 * Unauthenticated is deliberately NOT here: an access token that expired mid-flush is transient,
 * @fm/auth refreshes it, and marking the user's note "refused" for it would be a lie.
 */
function terminalRefusal(error: unknown): string | null {
  if (!(error instanceof ConnectError)) return null;
  switch (error.code) {
    case Code.PermissionDenied:
    case Code.NotFound:
    case Code.InvalidArgument:
    case Code.FailedPrecondition:
      return error.rawMessage.trim() || Code[error.code];
    default:
      return null;
  }
}
