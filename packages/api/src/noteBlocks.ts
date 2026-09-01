/**
 * Pure block helpers, used by the notes hooks and by the editor. They live outside notes.ts
 * for the same reason optimistic.ts lives outside hooks.ts: that module imports the React
 * provider, and these functions are the part that can actually be wrong, so they are kept
 * loadable — and therefore testable — without a React tree.
 *
 * Everything here is re-exported from notes.ts; the app imports it from @fm/api as usual.
 */

import type { Block, BlockType } from "@fm/sdk/notes/v1/notes_pb";

/** The block fields the local helpers need. A wire `Block` satisfies it. */
export type BlockLike = Pick<Block, "type" | "text"> & Partial<Pick<Block, "checked">>;

/**
 * The BlockType members this module needs, written as their wire numbers and typed as the
 * enum. The generated `BlockType` is a TypeScript `enum`, and importing it as a *value* would
 * make this module unloadable under `node --test`, which strips types rather than compiling
 * them. The numbers come from libs/proto/notes/v1/notes.proto and are fixed by the contract:
 * an enum value that changed number would already be a breaking wire change.
 */
const BLOCK_TODO: BlockType = 3;
const BLOCK_IMAGE: BlockType = 8;
const BLOCK_DIVIDER: BlockType = 9;

/** Blocks that carry no text of their own and must not leave blank lines in a flattening. */
const TEXTLESS_BLOCKS = new Set<BlockType>([BLOCK_DIVIDER, BLOCK_IMAGE]);

/**
 * Flattens a block array to plain text — the note's preview line, the string a share sheet
 * copies, the haystack a local "find in note" scans.
 *
 * Exported because the app needs the same flattening the server used for `note.preview`, and
 * re-implementing it in a screen is how the preview and the copy end up differing by a
 * newline. Dividers and images contribute nothing rather than an empty line.
 */
export function blocksToPlainText(blocks: readonly BlockLike[] | undefined): string {
  if (!blocks) return "";
  return blocks
    .filter((block) => !TEXTLESS_BLOCKS.has(block.type) && block.text.trim() !== "")
    .map((block) => block.text.trim())
    .join("\n");
}

/** The "6/9" a list row draws. */
export interface TaskCounts {
  total: number;
  done: number;
}

/**
 * Counts the TODO blocks in a note. `checked` is meaningless on every other block type (the
 * proto says it is false there), so only TODO blocks are looked at — a bullet that somehow
 * carries `checked: true` must not be counted as a finished task.
 *
 * The server sends `task_total`/`task_done` on list rows; this is the local answer for the
 * editor, which has edited blocks the server has not seen yet.
 */
export function countTasks(blocks: readonly BlockLike[] | undefined): TaskCounts {
  let total = 0;
  let done = 0;
  for (const block of blocks ?? []) {
    if (block.type !== BLOCK_TODO) continue;
    total += 1;
    if (block.checked) done += 1;
  }
  return { total, done };
}
