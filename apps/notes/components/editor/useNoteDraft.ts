import { Code } from "@connectrpc/connect";
import { useUpdateNote } from "@fm/api";
import type { Block, Note } from "@fm/sdk/notes/v1/notes_pb";
import { useCallback, useEffect, useRef, useState } from "react";

import { ensureBlocks } from "./blocks.ts";

/**
 * The editor's save loop.
 *
 * UpdateNote writes the WHOLE note — title plus the full block array — so there is nothing to
 * batch and nothing to diff: the only question is when to send it. The answer is a debounce,
 * because a keystroke-per-request editor on a phone network is a queue of requests that arrive
 * out of order and a save state that flickers.
 *
 * `expectedVersion` is what makes concurrent editing safe. Every write carries the version
 * this editor last read; the server refuses with ABORTED when someone else has moved on, and
 * this hook stops and surfaces the conflict rather than retrying with 0 and eating their
 * paragraph. Forcing the write (version 0) is a thing the USER does, from the banner.
 */

export const SAVE_DEBOUNCE_MS = 800;

export type SaveState =
  | "clean"
  | "dirty"
  | "saving"
  | "saved"
  /** The server refused: someone else saved a newer version. Nothing is sent until resolved. */
  | "conflict"
  | "error";

export interface NoteDraft {
  title: string;
  blocks: Block[];
  state: SaveState;
  /** True once the user has typed anything into this note in this session. */
  touched: boolean;
  setTitle: (title: string) => void;
  setBlocks: (next: Block[]) => void;
  /** Writes now instead of waiting out the debounce — leaving the screen, "Done", app blur. */
  flush: () => void;
  /**
   * Conflict resolution: throw the local edit away and take the server's note. Pass the
   * freshly refetched note — the copy this hook was rendered with is the stale one.
   */
  reload: (from?: Note) => void;
  /** Conflict resolution: force the write with expected_version 0, as the proto describes. */
  overwrite: () => void;
}

export function useNoteDraft(note: Note | null | undefined, canEdit: boolean): NoteDraft {
  const update = useUpdateNote();

  const [title, setTitleState] = useState("");
  const [blocks, setBlocksState] = useState<Block[]>(() => ensureBlocks([]));
  const [state, setStateRaw] = useState<SaveState>("clean");
  // Mirrored in a ref because the debounce timer and `flush` have to READ the current state
  // to decide whether to send. Reading it inside a `setState` updater would put a network
  // call in a function React is allowed to run twice.
  const stateRef = useRef<SaveState>("clean");
  const setState = useCallback((next: SaveState) => {
    stateRef.current = next;
    setStateRaw(next);
  }, []);
  const [touched, setTouched] = useState(false);

  /** The note this draft is currently holding, and the version its next write will claim. */
  const hydratedRef = useRef<string>("");
  const versionRef = useRef<bigint>(0n);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const latest = useRef({ title: "", blocks: [] as Block[], noteId: "" });

  latest.current = { title, blocks, noteId: note?.id ?? "" };

  const hydrate = useCallback(
    (from: Note) => {
      hydratedRef.current = from.id;
      versionRef.current = from.version;
      setTitleState(from.title);
      setBlocksState(ensureBlocks(from.blocks));
      setState("clean");
      setTouched(false);
    },
    [setState],
  );

  // Hydration happens once per note. A later refetch of the SAME note must not overwrite what
  // is on screen — that is the bug where a background refresh eats the sentence you are
  // halfway through. The conflict banner is how a newer server version gets in.
  useEffect(() => {
    if (note && note.id !== hydratedRef.current) hydrate(note);
  }, [note, hydrate]);

  const send = useCallback(
    (expectedVersion: bigint) => {
      const { noteId, title: t, blocks: b } = latest.current;
      if (noteId === "") return;
      setState("saving");
      update.mutate(
        { noteId, title: t, blocks: b, expectedVersion },
        {
          onSuccess: (saved) => {
            if (saved) versionRef.current = saved.version;
            setState("saved");
          },
          onError: (error) => {
            const code = (error as { code?: Code }).code;
            setState(code === Code.Aborted ? "conflict" : "error");
          },
        },
      );
    },
    [setState, update],
  );

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const schedule = useCallback(() => {
    if (!canEdit) return;
    // A conflict is a stop: nothing else is sent until the user picks reload or overwrite.
    if (stateRef.current === "conflict") return;
    clearTimer();
    setState("dirty");
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      if (stateRef.current === "conflict") return;
      send(versionRef.current);
    }, SAVE_DEBOUNCE_MS);
  }, [canEdit, clearTimer, send, setState]);

  const setTitle = useCallback(
    (next: string) => {
      setTitleState(next);
      setTouched(true);
      latest.current = { ...latest.current, title: next };
      schedule();
    },
    [schedule],
  );

  const setBlocks = useCallback(
    (next: Block[]) => {
      setBlocksState(next);
      setTouched(true);
      latest.current = { ...latest.current, blocks: next };
      schedule();
    },
    [schedule],
  );

  const flush = useCallback(() => {
    clearTimer();
    const current = stateRef.current;
    if (current === "dirty" || current === "error") send(versionRef.current);
  }, [clearTimer, send]);

  const reload = useCallback(
    (from?: Note) => {
      clearTimer();
      const source = from ?? note;
      if (source) hydrate(source);
    },
    [clearTimer, hydrate, note],
  );

  const overwrite = useCallback(() => {
    clearTimer();
    // 0 forces the write. The proto allows it precisely so the app can offer this AFTER
    // showing the conflict, never instead of showing it.
    send(0n);
  }, [clearTimer, send]);

  useEffect(() => clearTimer, [clearTimer]);

  return { title, blocks, state, touched, setTitle, setBlocks, flush, reload, overwrite };
}
