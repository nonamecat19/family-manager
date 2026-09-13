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


export type { QueuedNote, QueueRejection } from "./store.ts";
export { newClientId } from "./store.ts";

const RETRY_COOLDOWN_MS = 15_000;

function handle(ownerId: string): File | null {
  try {
    return new File(Paths.document, `capture-queue-${ownerFileKey(ownerId)}.json`);
  } catch {
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

let flushing = false;
let retryAfter = 0;

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

export async function resetCaptureQueue(): Promise<void> {
  retryAfter = 0;
  await store.reset();
}

export interface CaptureQueue {
  queued: QueuedNote[];
  pending: QueuedNote[];
  rejected: QueuedNote[];
  online: boolean;
  syncing: boolean;
  ownerId: string | null;
  capture: (note: Omit<QueuedNote, "clientId" | "capturedAt" | "rejection">) => Promise<void>;
  flush: () => Promise<void>;
  refile: (clientId: string, notebookId?: string) => Promise<void>;
}

export function useCaptureQueue(): CaptureQueue {
  const { status, getAccessToken } = useAuth();
  const [queued, setQueued] = useState<QueuedNote[]>(queueSnapshot);
  const [online, setOnline] = useState(true);
  const [syncing, setSyncing] = useState(flushing);
  const [ownerId, setOwnerId] = useState<string | null>(queueOwner);
  const createNote = useCreateNote();
  const createNoteRef = useRef(createNote);
  const ownerRef = useRef<string | null>(ownerId);

  useEffect(() => {
    createNoteRef.current = createNote;
  });

  useEffect(() => {
    ownerRef.current = ownerId;
  }, [ownerId]);

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
    setFlushing(true);
    try {
      await hydrateQueue(owner);
      if (queueOwner() !== owner) return;
      const pending = queueSnapshot().filter((item) => !item.rejection);
      if (pending.length === 0) return;
      for (const item of pending) {
        try {
          await createNoteRef.current.mutateAsync({
            notebookId: item.notebookId,
            title: item.title,
            blocks: item.blocks,
            clientId: item.clientId,
          });
        } catch (error) {
          const refusal = terminalRefusal(error);
          if (refusal === null) throw error;
          await store.reject(owner, item.clientId, { reason: refusal, at: Date.now() });
          continue;
        }
        await dequeueNote(owner, item.clientId);
      }
    } catch (error) {
      if (error instanceof QueueOwnerError) {
        return;
      }
      console.warn("[notes] capture queue flush stopped", error);
      retryAfter = Date.now() + RETRY_COOLDOWN_MS;
    } finally {
      setFlushing(false);
    }
  }, []);

  const capture = useCallback(
    async (note: Omit<QueuedNote, "clientId" | "capturedAt" | "rejection">) => {
      const owner = ownerRef.current;
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
