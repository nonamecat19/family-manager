import type { Block } from "@fm/sdk/notes/v1/notes_pb";

/**
 * The capture queue's state machine, with the file system held at arm's length.
 *
 * queue.ts owns the expo-file-system backing and the React binding; this module owns the rules
 * that are easy to get wrong and impossible to test through a component:
 *
 *  - A QUEUE BELONGS TO ONE USER. Every read, write and mutation is addressed to an owner id,
 *    the stored file names its owner, and a file whose owner does not match the person asking
 *    is NOT adopted — it is left where it is and the queue starts empty. This is the whole
 *    product promise on a shared device: a note user A captured offline and never flushed must
 *    not be created in user B's account because B was the next to sign in. The file name alone
 *    would not be enough (a restored backup, a reinstall, a file copied by hand); the owner is
 *    therefore verified from the CONTENTS as well.
 *  - hydration happens EXACTLY once per owner, and every caller awaits the same promise.
 *    Flipping a `hydrated` flag before awaiting the read opens a window where a second caller
 *    sees an empty queue, writes a file holding only its own note, and then has the in-flight
 *    read publish the old on-disk list over the top of it. The note is gone.
 *  - every mutation runs on one chain, after hydration. Two captures a few milliseconds apart
 *    would otherwise both read `items`, both append to the same base, and the second write
 *    would erase the first.
 *  - a mutation resolves only when the disk took it. Publishing optimistically and logging a
 *    failed write reports a note as captured when nothing was stored, and the sheet clears the
 *    user's text on that word. A failed write is a failed enqueue.
 *
 * Injecting the storage is what makes those rules testable under `node --test`: the backend is
 * two async string methods keyed by owner, not a native module.
 */

/** Why the server refused a queued note, kept so the app can say so instead of retrying forever. */
export interface QueueRejection {
  /** The server's own sentence, when it wrote one for a person. */
  reason: string;
  /** Epoch millis of the refusal. */
  at: number;
}

export interface QueuedNote {
  /** The idempotency key. Minted here, sent to the server, kept until the create succeeds. */
  clientId: string;
  notebookId: string;
  title: string;
  blocks: Block[];
  /** Epoch millis, so the list can sort a queued row in with the real ones. */
  capturedAt: number;
  /**
   * Set when a create was refused for a reason retrying cannot fix — capturing into a notebook
   * the user may only VIEW is the one that happens in practice. A rejected note stays in the
   * queue with the user's writing intact and is skipped by the flush loop, so one impossible
   * note cannot wedge the queue behind it; Settings offers to refile it.
   */
  rejection?: QueueRejection;
}

/**
 * What the queue needs from a disk, addressed by owner. `read` answers null when that owner has
 * written nothing yet.
 */
export interface QueueStorage {
  read: (ownerId: string) => Promise<string | null>;
  write: (ownerId: string, contents: string) => Promise<void>;
}

/** The on-disk shape. Version 1 was a bare array with no owner — it is never adopted. */
export interface QueueFile {
  version: 2;
  ownerId: string;
  notes: QueuedNote[];
}

export const QUEUE_FILE_VERSION = 2;

/** Thrown when a mutation is addressed to someone who is no longer the signed-in user. */
export class QueueOwnerError extends Error {
  constructor(expected: string, actual: string | null) {
    super(`capture queue is owned by ${actual ?? "nobody"}, not ${expected}`);
    this.name = "QueueOwnerError";
  }
}

export interface QueueStore {
  /** The current queue without waiting for the disk — the first render's value. */
  snapshot: () => QueuedNote[];
  /** Who the in-memory queue belongs to; null before the first hydrate and after a reset. */
  owner: () => string | null;
  subscribe: (listener: (next: QueuedNote[]) => void) => () => void;
  /**
   * Points the queue at `ownerId`'s file and awaits the single read of it. Safe to call
   * repeatedly and concurrently: every caller for the same owner waits on the same promise.
   * Called with a DIFFERENT owner it publishes an empty queue first, so no render can ever show
   * one account's captures under another account's name.
   *
   * It answers NOTHING, on purpose. It used to answer the queue, and that is a trap: the
   * promise is memoised, so its resolved VALUE is the array `items` pointed at when the read
   * finished, and every later mutation publishes a NEW array. A caller that read the queue out
   * of this promise was therefore looking at app-start content forever, and a note captured
   * after boot never appeared in it. Awaiting hydration and reading the queue are two
   * different questions — await this, then ask `snapshot()` what the queue holds NOW.
   */
  hydrate: (ownerId: string) => Promise<void>;
  /**
   * Forgets the queue in memory and the owner with it. THE FILE IS LEFT ON THE DISK: it holds
   * writing the user has not synced, it is named and stamped with their id, and only they can
   * ever hydrate it again. Sign-out must not double as "delete what I wrote".
   */
  reset: () => Promise<void>;
  /**
   * Appends a note, and resolves only once it is on the disk. A failed write REJECTS, so the
   * caller can keep the user's draft instead of clearing a note that was never saved.
   */
  enqueue: (ownerId: string, note: QueuedNote) => Promise<void>;
  /** Drops a note by client id. Rejects on a failed write, like `enqueue`. */
  dequeue: (ownerId: string, clientId: string) => Promise<void>;
  /** Records a refusal against a queued note, keeping the note. */
  reject: (ownerId: string, clientId: string, rejection: QueueRejection) => Promise<void>;
  /** Re-points a refused note at another notebook (or none) and clears the refusal. */
  refile: (ownerId: string, clientId: string, notebookId: string) => Promise<void>;
}

export function createQueueStore(storage: QueueStorage): QueueStore {
  let items: QueuedNote[] = [];
  let owner: string | null = null;
  let hydration: Promise<void> | null = null;
  // Bumped on every owner switch. A read that resolves after the switch belongs to nobody and
  // must not publish — that read is exactly how the previous user's notes would appear under
  // the new one.
  let generation = 0;
  // The tail of the mutation chain. Every mutation links onto it, so writes happen one at a
  // time and each one appends to the list the previous one produced.
  let tail: Promise<unknown> = Promise.resolve();
  const listeners = new Set<(next: QueuedNote[]) => void>();

  function publish(next: QueuedNote[]) {
    items = next;
    for (const listener of listeners) listener(items);
  }

  function requireOwner(ownerId: string) {
    if (owner !== ownerId) throw new QueueOwnerError(ownerId, owner);
  }

  /**
   * Writes the list, and publishes it only once the write succeeded.
   *
   * The order is the whole point. Publishing first and logging the failure let `enqueue`
   * RESOLVE when nothing had reached the disk — and CaptureSheet clears the user's text on a
   * resolved capture, so a full or read-only disk silently ate the note. Letting the write
   * throw makes that a rejected enqueue, and leaves `items` describing what is actually
   * stored rather than what we hoped would be.
   */
  async function persist(ownerId: string, next: QueuedNote[]): Promise<void> {
    requireOwner(ownerId);
    const file: QueueFile = { version: QUEUE_FILE_VERSION, ownerId, notes: next };
    await storage.write(ownerId, JSON.stringify(file));
    // Signed out while the write was in flight: the bytes were still theirs to keep, but the
    // list on screen now belongs to nobody.
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
        // A version-1 file (a bare array) lands here. It names no owner, so nobody can prove
        // it is theirs — it is left alone rather than adopted or deleted.
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
    // Synchronously, before any await: whatever was on screen belonged to someone else.
    publish([]);
    hydration = load(ownerId, generation);
    return hydration;
  }

  /** Runs `work` after everything already queued, whether that finished or threw. */
  function serialise<T>(work: () => Promise<T>): Promise<T> {
    const run = tail.then(work, work);
    tail = run.then(
      () => undefined,
      () => undefined,
    );
    return run;
  }

  /** The body every mutation shares: prove the owner, wait for their disk, prove it again. */
  function mutate(ownerId: string, next: (current: QueuedNote[]) => QueuedNote[]): Promise<void> {
    return serialise(async () => {
      // Checked BEFORE hydrate, which would otherwise switch the queue back to a user who
      // signed out while this mutation was waiting its turn on the chain.
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

/** Narrows a parsed file to the owned shape. A bare array — version 1 — deliberately fails. */
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

/** A client id that is unique per capture and stable across retries of that capture. */
export function newClientId(): string {
  return `c${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`;
}
