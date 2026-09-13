import { create } from "@bufbuild/protobuf";
import { BlockSchema, BlockType, type Block } from "@fm/sdk/notes/v1/notes_pb";


export const NO_BLOCK = -1;

export function canRetype(blocks: readonly Block[], index: number): boolean {
  const block = blocks[index];
  return block !== undefined && !HOLDS_NO_TEXT.has(block.type);
}

let counter = 0;

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

export function ensureBlocks(blocks: readonly Block[]): Block[] {
  return blocks.length > 0 ? [...blocks] : [makeBlock()];
}

export function replaceAt(blocks: readonly Block[], index: number, next: Block): Block[] {
  const out = [...blocks];
  out[index] = next;
  return out;
}

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

export function insertAfter(blocks: readonly Block[], index: number): { blocks: Block[]; focus: number } {
  const out = [...blocks];
  const at = Math.min(Math.max(index + 1, 0), out.length);
  out.splice(at, 0, makeBlock());
  return { blocks: out, focus: at };
}

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

export function insertDivider(blocks: readonly Block[], index: number): { blocks: Block[]; focus: number } {
  const source = blocks[index];
  const out = [...blocks];
  const rule = makeBlock({ type: BlockType.DIVIDER });

  if (source && source.type === BlockType.PARAGRAPH && source.text === "") {
    out[index] = { ...rule, id: source.id };
    out.splice(index + 1, 0, makeBlock());
    return { blocks: out, focus: index + 1 };
  }

  const at = Math.min(Math.max(index + 1, 0), out.length);
  out.splice(at, 0, rule, makeBlock());
  return { blocks: out, focus: at + 1 };
}

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

const HOLDS_NO_TEXT: ReadonlySet<BlockType> = new Set([BlockType.DIVIDER, BlockType.IMAGE]);

const CARRIES_TO_NEXT_LINE: ReadonlySet<BlockType> = new Set([
  BlockType.TODO,
  BlockType.BULLET,
  BlockType.NUMBERED,
  BlockType.QUOTE,
  BlockType.CODE,
]);

export function outline(blocks: readonly Block[]): { id: string; text: string; level: number }[] {
  return blocks
    .filter((b) => b.type === BlockType.HEADING && b.text.trim() !== "")
    .map((b) => ({ id: b.id, text: b.text.trim(), level: b.level || 1 }));
}

export function isEmptyDocument(title: string, blocks: readonly Block[]): boolean {
  if (title.trim() !== "") return false;
  return blocks.every((b) => b.text.trim() === "" && b.imageUrl === "");
}
