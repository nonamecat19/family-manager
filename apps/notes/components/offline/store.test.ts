import assert from "node:assert/strict";
import test from "node:test";

import {
  QUEUE_FILE_VERSION,
  createQueueStore,
  type QueueFile,
  type QueuedNote,
  type QueueStorage,
} from "./store.ts";

/**
 * The queue's ownership and ordering rules, tested without React Native: store.ts takes its
 * storage as two async string methods keyed by owner, so a slow in-memory disk is enough to
 * reproduce both the races and the shared-tablet leak.
 *
 * Run with `node --test apps/notes/components/offline/store.test.ts` (Node strips the types).
 */

const ALICE = "user-alice";
const BOB = "user-bob";

function note(clientId: string): QueuedNote {
  return { clientId, notebookId: "", title: clientId, blocks: [], capturedAt: 0 };
}

function fileFor(ownerId: string, notes: QueuedNote[]): string {
  const file: QueueFile = { version: QUEUE_FILE_VERSION, ownerId, notes };
  return JSON.stringify(file);
}

/**
 * A disk whose read takes `delayMs`, so a second caller can land inside the read window. It is
 * keyed by owner, exactly like the real one — the file name carries the user id.
 */
function slowStorage(initial: Record<string, string> = {}, delayMs = 20) {
  const files: Record<string, string> = { ...initial };
  const storage: QueueStorage = {
    read: async (ownerId) => {
      await new Promise((resolve) => setTimeout(resolve, delayMs));
      return files[ownerId] ?? null;
    },
    write: async (ownerId, next) => {
      files[ownerId] = next;
    },
  };
  return {
    storage,
    files,
    written: (ownerId: string) => {
      const raw = files[ownerId];
      if (raw === undefined) return [];
      return (JSON.parse(raw) as QueueFile).notes;
    },
  };
}

const ids = (notes: readonly QueuedNote[]) => notes.map((item) => item.clientId);

test("an enqueue that lands inside the hydrate window keeps both notes", async () => {
  const disk = slowStorage({ [ALICE]: fileFor(ALICE, [note("on-disk")]) });
  const store = createQueueStore(disk.storage);

  // The race: hydrate is in flight (its read has not resolved) when the capture arrives.
  const hydrating = store.hydrate(ALICE);
  await store.enqueue(ALICE, note("captured"));
  await hydrating;

  assert.deepEqual(ids(store.snapshot()), ["on-disk", "captured"]);
  assert.deepEqual(ids(disk.written(ALICE)), ["on-disk", "captured"]);
});

test("concurrent enqueues do not overwrite each other", async () => {
  const disk = slowStorage();
  const store = createQueueStore(disk.storage);

  store.hydrate(ALICE);
  await Promise.all([
    store.enqueue(ALICE, note("a")),
    store.enqueue(ALICE, note("b")),
    store.enqueue(ALICE, note("c")),
  ]);

  assert.deepEqual(ids(disk.written(ALICE)), ["a", "b", "c"]);
});

test("hydrate reads the storage exactly once however many callers ask", async () => {
  let reads = 0;
  const store = createQueueStore({
    read: async (ownerId) => {
      reads += 1;
      await new Promise((resolve) => setTimeout(resolve, 10));
      return fileFor(ownerId, [note("on-disk")]);
    },
    write: async () => undefined,
  });

  await Promise.all([store.hydrate(ALICE), store.hydrate(ALICE), store.hydrate(ALICE)]);

  assert.equal(reads, 1);
  assert.equal(store.snapshot().length, 1);
  await store.hydrate(ALICE);
  assert.equal(reads, 1);
});

test("hydrate answers nothing, so no caller can be left holding a stale queue", async () => {
  const disk = slowStorage({ [ALICE]: fileFor(ALICE, [note("on-disk")]) });
  const store = createQueueStore(disk.storage);

  await store.hydrate(ALICE);
  await store.enqueue(ALICE, note("captured-after-boot"));

  // The regression this guards: hydrate() is memoised, and it used to RESOLVE TO the `items`
  // array as it stood when the read finished. Every mutation publishes a NEW array, so that
  // resolved value stayed frozen at app-start content forever — a flush that built its work
  // list from `await hydrateQueue()` never saw a note captured after boot, and that note sat
  // on the disk and never synced. Awaiting hydration and reading the queue are two questions,
  // and only `snapshot()` answers the second one.
  const answer: unknown = await store.hydrate(ALICE);
  assert.equal(answer, undefined);

  assert.deepEqual(ids(store.snapshot()), ["on-disk", "captured-after-boot"]);
});

test("an enqueue whose write fails rejects instead of reporting the note as saved", async () => {
  let contents: string | null = null;
  let failing = true;
  const store = createQueueStore({
    read: async () => contents,
    write: async (_ownerId, next) => {
      if (failing) throw new Error("ENOSPC: no space left on device");
      contents = next;
    },
  });

  store.hydrate(ALICE);

  // Resolving here is what let CaptureSheet clear the user's text for a note that reached no
  // disk. A rejection is the only thing that lets the sheet keep the draft.
  await assert.rejects(store.enqueue(ALICE, note("would-be-lost")), /ENOSPC/);
  assert.deepEqual(store.snapshot(), []);
  assert.equal(contents, null);

  // The mutation chain survives a rejected write: the next capture still gets through.
  failing = false;
  await store.enqueue(ALICE, note("kept"));
  assert.deepEqual(ids(store.snapshot()), ["kept"]);
  assert.deepEqual(ids((JSON.parse(contents ?? "null") as QueueFile).notes), ["kept"]);
});

test("a dequeue after a capture leaves the other queued notes alone", async () => {
  const disk = slowStorage({ [ALICE]: fileFor(ALICE, [note("on-disk")]) });
  const store = createQueueStore(disk.storage);

  store.hydrate(ALICE);
  await store.enqueue(ALICE, note("captured"));
  await store.dequeue(ALICE, "on-disk");

  assert.deepEqual(ids(disk.written(ALICE)), ["captured"]);
});

/* ------------------------------------------------------- one device, two people */

test("one user's queue is never adopted by the next user to sign in", async () => {
  // The leak this replaces: a single unscoped capture-queue.json, so a note Alice wrote offline
  // and never flushed was created in Bob's account the moment Bob signed in on the tablet.
  const disk = slowStorage({ [ALICE]: fileFor(ALICE, [note("alices-unsent-note")]) });
  const store = createQueueStore(disk.storage);

  await store.hydrate(ALICE);
  assert.deepEqual(ids(store.snapshot()), ["alices-unsent-note"]);

  await store.reset();
  await store.hydrate(BOB);

  assert.deepEqual(store.snapshot(), []);
  assert.equal(store.owner(), BOB);
  // And Alice's file is untouched — signing out is not "delete what I wrote".
  assert.deepEqual(ids(disk.written(ALICE)), ["alices-unsent-note"]);
});

test("a file stamped with another owner is refused even under the right name", async () => {
  // Belt and braces for a restored backup, a reinstall, or a file copied by hand: the name says
  // Bob, the contents say Alice, and the contents win.
  const disk = slowStorage({ [BOB]: fileFor(ALICE, [note("not-bobs")]) });
  const store = createQueueStore(disk.storage);

  await store.hydrate(BOB);

  assert.deepEqual(store.snapshot(), []);
});

test("the old unscoped file — a bare array, no owner — is never adopted", async () => {
  const disk = slowStorage({ [BOB]: JSON.stringify([note("from-the-old-queue")]) });
  const store = createQueueStore(disk.storage);

  await store.hydrate(BOB);

  assert.deepEqual(store.snapshot(), []);
});

test("signing back in gets the unsent notes back", async () => {
  const disk = slowStorage();
  const store = createQueueStore(disk.storage);

  await store.hydrate(ALICE);
  await store.enqueue(ALICE, note("written-on-the-train"));
  await store.reset();
  assert.deepEqual(store.snapshot(), []);
  assert.equal(store.owner(), null);

  await store.hydrate(ALICE);
  assert.deepEqual(ids(store.snapshot()), ["written-on-the-train"]);
});

test("a mutation addressed to a user who has signed out is refused", async () => {
  const disk = slowStorage();
  const store = createQueueStore(disk.storage);

  await store.hydrate(ALICE);
  await store.hydrate(BOB);

  await assert.rejects(store.enqueue(ALICE, note("late")), /capture queue is owned by/);
  assert.deepEqual(store.snapshot(), []);
  assert.equal(disk.files[ALICE], undefined);
});

test("a read that resolves after the owner changed does not publish", async () => {
  const disk = slowStorage(
    { [ALICE]: fileFor(ALICE, [note("alices")]), [BOB]: fileFor(BOB, [note("bobs")]) },
    20,
  );
  const store = createQueueStore(disk.storage);

  const alices = store.hydrate(ALICE);
  const bobs = store.hydrate(BOB);
  await Promise.all([alices, bobs]);

  assert.deepEqual(ids(store.snapshot()), ["bobs"]);
});

/* ------------------------------------------------------------ refused captures */

test("a refused note keeps its writing and steps out of the retry loop", async () => {
  const disk = slowStorage();
  const store = createQueueStore(disk.storage);

  await store.hydrate(ALICE);
  await store.enqueue(ALICE, { ...note("view-only-notebook"), notebookId: "nb-shared" });
  await store.reject(ALICE, "view-only-notebook", { reason: "you can only view that", at: 7 });

  const [refused] = store.snapshot();
  assert.equal(refused?.title, "view-only-notebook");
  assert.equal(refused?.rejection?.reason, "you can only view that");
  // Still on the disk: the user's words are not what gets thrown away when a server says no.
  assert.equal(disk.written(ALICE).length, 1);

  await store.refile(ALICE, "view-only-notebook", "");
  const [refiled] = store.snapshot();
  assert.equal(refiled?.notebookId, "");
  assert.equal(refiled?.rejection, undefined);
  assert.equal(refiled?.clientId, "view-only-notebook");
});
