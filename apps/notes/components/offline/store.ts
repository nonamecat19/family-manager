import type { Block } from "@fm/sdk/notes/v1/notes_pb";


export interface QueueRejection {
  reason: string;
  at: number;
}

export interface QueuedNote {
  clientId: string;
  notebookId: string;
  title: string;
  blocks: Block[];
  capturedAt: number;
  rejection?: QueueRejection;
}

export interface QueueStorage {
  read: (ownerId: string) => Promise<string | null>;
  write: (ownerId: string, contents: string) => Promise<void>;
}

export interface QueueFile {
  version: 2;
  ownerId: string;
  notes: QueuedNote[];
}

export const QUEUE_FILE_VERSION = 2;

export class QueueOwnerError extends Error {
  constructor(expected: string, actual: string | null) {
    super(`capture queue is owned by ${actual ?? "nobody"}, not ${expected}`);
    this.name = "QueueOwnerError";
  }
}

export interface QueueStore {
  snapshot: () => QueuedNote[];
  owner: () => string | null;
  subscribe: (listener: (next: QueuedNote[]) => void) => () => void;
  hydrate: (ownerId: string) => Promise<void>;
  reset: () => Promise<void>;
  enqueue: (ownerId: string, note: QueuedNote) => Promise<void>;
  dequeue: (ownerId: string, clientId: string) => Promise<void>;
  reject: (ownerId: string, clientId: string, rejection: QueueRejection) => Promise<void>;
  refile: (ownerId: string, clientId: string, notebookId: string) => Promise<void>;
}

export function createQueueStore(storage: QueueStorage): QueueStore {
  let items: QueuedNote[] = [];
  let owner: string | null = null;
  let hydration: Promise<void> | null = null;
  let generation = 0;
  let tail: Promise<unknown> = Promise.resolve();
  const listeners = new Set<(next: QueuedNote[]) => void>();

  function publish(next: QueuedNote[]) {
    items = next;
    for (const listener of listeners) listener(items);
  }

  function requireOwner(ownerId: string) {
    if (owner !== ownerId) throw new QueueOwnerError(ownerId, owner);
  }

  async function persist(ownerId: string, next: QueuedNote[]): Promise<void> {
    requireOwner(ownerId);
    const file: QueueFile = { version: QUEUE_FILE_VERSION, ownerId, notes: next };
    await storage.write(ownerId, JSON.stringify(file));
    if (owner !== ownerId) return;
    publish(next);
  }

  async function load(ownerId: string, forGeneration: number): Promise<void> {
    try {
      const contents = await storage.read(ownerId);
      if (forGeneration !== generation) return;
      if (contents === null) return;
      const parsed: unknown = JSON.parse(contents);
      const file = asQueueFile(parsed);
      if (!file) {
        console.warn("[notes] ignoring a capture queue file this app cannot attribute");
        return;
      }
      if (file.ownerId !== ownerId) {
        console.warn("[notes] capture queue file belongs to another account — not adopted");
        return;
      }
      publish(file.notes);
    } catch (error) {
      console.warn("[notes] could not read the capture queue", error);
    }
  }

  function hydrate(ownerId: string): Promise<void> {
    if (owner === ownerId && hydration) return hydration;
    owner = ownerId;
    generation += 1;
    publish([]);
    hydration = load(ownerId, generation);
    return hydration;
  }

  function serialise<T>(work: () => Promise<T>): Promise<T> {
    const run = tail.then(work, work);
    tail = run.then(
      () => undefined,
      () => undefined,
    );
    return run;
  }

  function mutate(ownerId: string, next: (current: QueuedNote[]) => QueuedNote[]): Promise<void> {
    return serialise(async () => {
      requireOwner(ownerId);
      await hydrate(ownerId);
      requireOwner(ownerId);
      await persist(ownerId, next(items));
    });
  }

  return {
    snapshot: () => items,
    owner: () => owner,
    subscribe(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    hydrate,
    reset: () =>
      serialise(async () => {
        owner = null;
        hydration = null;
        generation += 1;
        publish([]);
      }),
    enqueue: (ownerId, note) => mutate(ownerId, (current) => [...current, note]),
    dequeue: (ownerId, clientId) =>
      mutate(ownerId, (current) => current.filter((item) => item.clientId !== clientId)),
    reject: (ownerId, clientId, rejection) =>
      mutate(ownerId, (current) =>
        current.map((item) => (item.clientId === clientId ? { ...item, rejection } : item)),
      ),
    refile: (ownerId, clientId, notebookId) =>
      mutate(ownerId, (current) =>
        current.map((item) =>
          item.clientId === clientId
            ? { clientId: item.clientId, notebookId, title: item.title, blocks: item.blocks, capturedAt: item.capturedAt }
            : item,
        ),
      ),
  };
}

export function asQueueFile(value: unknown): QueueFile | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) return null;
  const candidate = value as Partial<QueueFile>;
  if (candidate.version !== QUEUE_FILE_VERSION) return null;
  if (typeof candidate.ownerId !== "string" || candidate.ownerId === "") return null;
  if (!Array.isArray(candidate.notes)) return null;
  return {
    version: QUEUE_FILE_VERSION,
    ownerId: candidate.ownerId,
    notes: candidate.notes.filter(isQueuedNote),
  };
}

export function isQueuedNote(value: unknown): value is QueuedNote {
  if (typeof value !== "object" || value === null) return false;
  const candidate = value as Partial<QueuedNote>;
  return (
    typeof candidate.clientId === "string" &&
    typeof candidate.title === "string" &&
    Array.isArray(candidate.blocks)
  );
}

export function newClientId(): string {
  return `c${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`;
}
