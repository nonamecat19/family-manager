import type { Block, BlockType } from "@fm/sdk/notes/v1/notes_pb";

export type BlockLike = Pick<Block, "type" | "text"> & Partial<Pick<Block, "checked">>;

const BLOCK_TODO: BlockType = 3;
const BLOCK_IMAGE: BlockType = 8;
const BLOCK_DIVIDER: BlockType = 9;

const TEXTLESS_BLOCKS = new Set<BlockType>([BLOCK_DIVIDER, BLOCK_IMAGE]);

export function blocksToPlainText(blocks: readonly BlockLike[] | undefined): string {
  if (!blocks) return "";
  return blocks
    .filter((block) => !TEXTLESS_BLOCKS.has(block.type) && block.text.trim() !== "")
    .map((block) => block.text.trim())
    .join("\n");
}

export interface TaskCounts {
  total: number;
  done: number;
}

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
