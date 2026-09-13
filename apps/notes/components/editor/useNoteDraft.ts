import { Code } from "@connectrpc/connect";
import { useUpdateNote } from "@fm/api";
import type { Block, Note } from "@fm/sdk/notes/v1/notes_pb";
import { useCallback, useEffect, useRef, useState } from "react";

import { ensureBlocks } from "./blocks.ts";


export const SAVE_DEBOUNCE_MS = 800;

export type SaveState =
  | "clean"
  | "dirty"
  | "saving"
  | "saved"
  | "conflict"
  | "error";

export interface NoteDraft {
  title: string;
  blocks: Block[];
  state: SaveState;
  touched: boolean;
  setTitle: (title: string) => void;
  setBlocks: (next: Block[]) => void;
  flush: () => void;
  reload: (from?: Note) => void;
  overwrite: () => void;
}

export function useNoteDraft(note: Note | null | undefined, canEdit: boolean): NoteDraft {
  const update = useUpdateNote();

  const [title, setTitleState] = useState("");
  const [blocks, setBlocksState] = useState<Block[]>(() => ensureBlocks([]));
  const [state, setStateRaw] = useState<SaveState>("clean");
  const stateRef = useRef<SaveState>("clean");
  const setState = useCallback((next: SaveState) => {
    stateRef.current = next;
    setStateRaw(next);
  }, []);
  const [touched, setTouched] = useState(false);

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
    send(0n);
  }, [clearTimer, send]);

  useEffect(() => clearTimer, [clearTimer]);

  return { title, blocks, state, touched, setTitle, setBlocks, flush, reload, overwrite };
}
