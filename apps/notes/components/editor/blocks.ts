import { create } from "@bufbuild/protobuf";
import { BlockSchema, BlockType, type Block } from "@fm/sdk/notes/v1/notes_pb";

/**
 * The block document, as pure functions over `Block[]`.
 *
 * The editor component owns focus and text input; this file owns what a keystroke MEANS to
 * the document — split, merge, retype, markdown entry. Keeping them apart is what makes the
 * behaviour readable: none of these touch React, and the component never rewrites an array
 * in place.
 *
 * Block ids are minted here and are stable across edits, because the proto says the server
 * never rewrites a block array — it stores the one it is given, and a comment or a cursor
 * points at an id that has to survive the note being re-saved.
 */

/**
 * "No block is focused" — where the caret index starts, and where it goes while the caret is
 * in the title.
 *
 * It has to be a value no index can ever be. 0 is a real block, so a screen that starts at 0
 * is a screen where the first press on the block bar retypes the note's first block although
 * the user never put a caret in it. That is worst when the first block is an IMAGE: an image
 * block draws no TextInput, so it can never report focus and never correct the guess, and the
 * retype replaces it with an empty paragraph — a block the user cannot get back, from one tap.
 */
export const NO_BLOCK = -1;

/**
 * Can the block bar retype the block at `index`?
 *
 * False for NO_BLOCK and any index outside the array — nothing is focused, so there is nothing
 * to retype. False as well for the types that hold no text: an IMAGE or a DIVIDER draws no
 * TextInput, so it cannot be focused deliberately, but an index can still come to rest on one
 * (the gutter's move keeps the focused index and changes what lives at it), and retyping one
 * silently destroys what it held.
 */
export function canRetype(blocks: readonly Block[], index: number): boolean {
  const block = blocks[index];
  return block !== undefined && !HOLDS_NO_TEXT.has(block.type);
}

let counter = 0;

/** A client-minted, collision-resistant block id. `crypto.randomUUID` is not on every RN runtime. */
export function newBlockId(): string {
  counter += 1;
  return `b${Date.now().toString(36)}${counter.toString(36)}${Math.random().toString(36).slice(2, 8)}`;
}

export function makeBlock(partial: Partial<Omit<Block, "$typeName">> = {}): Block {
  return create(BlockSchema, {
    id: partial.id ?? newBlockId(),
    type: partial.type ?? BlockType.PARAGRAPH,
    text: partial.text ?? "",
    checked: partial.checked ?? false,
    level: partial.level ?? 0,
    imageUrl: partial.imageUrl ?? "",
    language: partial.language ?? "",
  });
}

/** A note always has at least one block to put the cursor in. */
export function ensureBlocks(blocks: readonly Block[]): Block[] {
  return blocks.length > 0 ? [...blocks] : [makeBlock()];
}

export function replaceAt(blocks: readonly Block[], index: number, next: Block): Block[] {
  const out = [...blocks];
  out[index] = next;
  return out;
}

/**
 * Enter. Everything left of the caret stays, everything right of it becomes a new block below.
 * The new block inherits list-ness — pressing Enter in a task list gives you another task —
 * but never inherits a heading, which is the one type where the next line is ordinary prose.
 */
export function splitAt(blocks: readonly Block[], index: number, offset: number): {
  blocks: Block[];
  focus: number;
} {
  const block = blocks[index];
  if (!block) return { blocks: [...blocks], focus: index };
  const head = block.text.slice(0, offset);
  const tail = block.text.slice(offset);
  const carried = CARRIES_TO_NEXT_LINE.has(block.type) ? block.type : BlockType.PARAGRAPH;
  const next = makeBlock({
    type: carried,
    text: tail,
    level: carried === block.type ? block.level : 0,
  });
  const out = [...blocks];
  out[index] = { ...block, text: head };
  out.splice(index + 1, 0, next);
  return { blocks: out, focus: index + 1 };
}

/**
 * Backspace at offset 0. A block that carries formatting loses the formatting first — the
 * same escape hatch every editor has, and the reason you can back out of a task without
 * deleting the words you already typed. A plain paragraph merges into the one above it, and
 * the caret lands where the join happened.
 *
 * There is nothing to merge INTO a block that holds no text. A DIVIDER or an IMAGE has no
 * TextInput to put the caret in, so joining a paragraph into one used to hide the paragraph's
 * words inside a `text` field nothing renders and drop the paragraph itself — text loss that
 * looked like a deletion the user asked for. It is a no-op instead. This matters more now that
 * image blocks are read-only leftovers (see BlockEditor): backspacing under one must not
 * rewrite it, and must not eat the sentence being written beside it.
 */
export function mergeBackwards(blocks: readonly Block[], index: number): {
  blocks: Block[];
  focus: number;
  offset: number;
} {
  const block = blocks[index];
  if (!block) return { blocks: [...blocks], focus: index, offset: 0 };

  if (block.type !== BlockType.PARAGRAPH) {
    return {
      blocks: replaceAt(blocks, index, { ...block, type: BlockType.PARAGRAPH, level: 0, checked: false }),
      focus: index,
      offset: 0,
    };
  }
  if (index === 0) return { blocks: [...blocks], focus: 0, offset: 0 };

  const previous = blocks[index - 1];
  if (!previous) return { blocks: [...blocks], focus: index, offset: 0 };
  if (HOLDS_NO_TEXT.has(previous.type)) return { blocks: [...blocks], focus: index, offset: 0 };
  const offset = previous.text.length;
  const out = [...blocks];
  out[index - 1] = { ...previous, text: previous.text + block.text };
  out.splice(index, 1);
  return { blocks: out, focus: index - 1, offset };
}

/**
 * The gutter's insert button: a fresh paragraph directly below `index`, and the caret in it.
 * Enter already does this from inside the text, but a DIVIDER or an IMAGE block has no text
 * to press Enter in — without this there is no way to write after one.
 */
export function insertAfter(blocks: readonly Block[], index: number): { blocks: Block[]; focus: number } {
  const out = [...blocks];
  const at = Math.min(Math.max(index + 1, 0), out.length);
  out.splice(at, 0, makeBlock());
  return { blocks: out, focus: at };
}

/** The gutter's up/down. Out-of-range moves are no-ops, so the caller can bind both ends. */
export function moveBlock(blocks: readonly Block[], index: number, delta: number): { blocks: Block[]; focus: number } {
  const to = index + delta;
  if (index < 0 || index >= blocks.length || to < 0 || to >= blocks.length) {
    return { blocks: [...blocks], focus: index };
  }
  const out = [...blocks];
  const [moved] = out.splice(index, 1);
  if (!moved) return { blocks: [...blocks], focus: index };
  out.splice(to, 0, moved);
  return { blocks: out, focus: to };
}

/**
 * A DIVIDER is the one block type that cannot be *retyped* into: it holds no text, so turning
 * the focused block into one would strand the caret in a block with no input. It is inserted
 * instead — the rule below the current block, and a paragraph after it to keep writing in.
 */
export function insertDivider(blocks: readonly Block[], index: number): { blocks: Block[]; focus: number } {
  const source = blocks[index];
  const out = [...blocks];
  const rule = makeBlock({ type: BlockType.DIVIDER });

  // An empty paragraph BECOMES the rule rather than leaving a blank line above it.
  if (source && source.type === BlockType.PARAGRAPH && source.text === "") {
    out[index] = { ...rule, id: source.id };
    out.splice(index + 1, 0, makeBlock());
    return { blocks: out, focus: index + 1 };
  }

  const at = Math.min(Math.max(index + 1, 0), out.length);
  out.splice(at, 0, rule, makeBlock());
  return { blocks: out, focus: at + 1 };
}

/**
 * Retype the focused block. Neither end of a retype may be a block that holds no text, and the
 * refusal lives here rather than only in the caller, the way the backspace-merge refusal does.
 *
 * Retyping OUT of a DIVIDER or an IMAGE destroys it — an image block's url is not shown
 * anywhere, so the paragraph it becomes is empty and there is nothing to undo it with. Retyping
 * INTO one strands the caret in a block with no input and hides the words that were in it. A
 * rule is inserted instead (`insertDivider`), and v1 offers no way to make an image at all.
 *
 * `index` may be NO_BLOCK or out of range; that is a no-op like every other out-of-range call
 * in this file.
 */
export function setType(blocks: readonly Block[], index: number, type: BlockType): Block[] {
  const block = blocks[index];
  if (!block) return [...blocks];
  if (HOLDS_NO_TEXT.has(block.type) || HOLDS_NO_TEXT.has(type)) return [...blocks];
  const level = type === BlockType.HEADING ? Math.max(1, block.level || 2) : 0;
  return replaceAt(blocks, index, {
    ...block,
    type,
    level,
    checked: type === BlockType.TODO ? block.checked : false,
  });
}

export function toggleChecked(blocks: readonly Block[], index: number): Block[] {
  const block = blocks[index];
  if (!block || block.type !== BlockType.TODO) return [...blocks];
  return replaceAt(blocks, index, { ...block, checked: !block.checked });
}

/**
 * Markdown-ish entry: the prefixes the design's editor accepts. Returns the retyped block, or
 * null when the text is not a prefix — the caller keeps the plain edit in that case.
 *
 * Only ever applied to a PARAGRAPH: typing "- " inside a quote is a hyphen, not a list.
 */
export function applyMarkdownPrefix(block: Block, text: string): Block | null {
  if (block.type !== BlockType.PARAGRAPH) return null;
  for (const rule of PREFIXES) {
    const match = rule.pattern.exec(text);
    if (!match) continue;
    return {
      ...block,
      type: rule.type,
      level: rule.level ?? 0,
      language: rule.type === BlockType.CODE ? block.language : "",
      text: text.slice(match[0].length),
    };
  }
  return null;
}

interface PrefixRule {
  pattern: RegExp;
  type: BlockType;
  level?: number;
}

const PREFIXES: readonly PrefixRule[] = [
  { pattern: /^### /, type: BlockType.HEADING, level: 3 },
  { pattern: /^## /, type: BlockType.HEADING, level: 2 },
  { pattern: /^# /, type: BlockType.HEADING, level: 1 },
  { pattern: /^\[[ xX]?\] /, type: BlockType.TODO },
  { pattern: /^[-*] /, type: BlockType.BULLET },
  { pattern: /^\d+\. /, type: BlockType.NUMBERED },
  { pattern: /^> /, type: BlockType.QUOTE },
  { pattern: /^```/, type: BlockType.CODE },
];

/**
 * The types the editor draws with no TextInput in them. Nothing can be merged into one, and
 * `canRetype` refuses to retype one — both are the same fact: the block holds no text, so any
 * edit that treats it as a line of prose loses what it does hold.
 */
const HOLDS_NO_TEXT: ReadonlySet<BlockType> = new Set([BlockType.DIVIDER, BlockType.IMAGE]);

/** The types where Enter gives you another one of the same. */
const CARRIES_TO_NEXT_LINE: ReadonlySet<BlockType> = new Set([
  BlockType.TODO,
  BlockType.BULLET,
  BlockType.NUMBERED,
  BlockType.QUOTE,
  BlockType.CODE,
]);

/** The headings the editor's "In this note" rail lists. */
export function outline(blocks: readonly Block[]): { id: string; text: string; level: number }[] {
  return blocks
    .filter((b) => b.type === BlockType.HEADING && b.text.trim() !== "")
    .map((b) => ({ id: b.id, text: b.text.trim(), level: b.level || 1 }));
}

/** Blocks that hold no text and no image — what an empty note actually looks like. */
export function isEmptyDocument(title: string, blocks: readonly Block[]): boolean {
  if (title.trim() !== "") return false;
  return blocks.every((b) => b.text.trim() === "" && b.imageUrl === "");
}
