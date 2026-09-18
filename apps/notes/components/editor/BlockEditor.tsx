import { BlockType, type Block } from "@fm/sdk/notes/v1/notes_pb";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Pressable, Text, TextInput, View } from "react-native";

import { Divider, Icon, organic, type IconName } from "@fm/ui";
import { strings } from "../i18n/index.ts";
import {
  NO_BLOCK,
  applyMarkdownPrefix,
  ensureBlocks,
  insertAfter,
  mergeBackwards,
  moveBlock,
  replaceAt,
  splitAt,
  toggleChecked,
} from "./blocks.ts";


export const BLOCK_COLUMN_WIDTH = 560;
export const GUTTER_LANE = 52;

export interface BlockEditorProps {
  title: string;
  onChangeTitle: (title: string) => void;
  blocks: Block[];
  onChangeBlocks: (blocks: Block[]) => void;
  editable: boolean;
  focused: number;
  onFocusedChange: (index: number) => void;
  size?: "compact" | "roomy";
}

interface PendingFocus {
  index: number;
  offset: number;
  nonce: number;
}

export function BlockEditor({
  title,
  onChangeTitle,
  blocks,
  onChangeBlocks,
  editable,
  focused,
  onFocusedChange,
  size = "roomy",
}: BlockEditorProps) {
  const [pending, setPending] = useState<PendingFocus | null>(null);
  const nonce = useRef(0);

  const request = (index: number, offset: number) => {
    nonce.current += 1;
    setPending({ index, offset, nonce: nonce.current });
    onFocusedChange(index);
  };

  const handleText = (index: number, next: string) => {
    const block = blocks[index];
    if (!block) return;

    const newline = next.indexOf("\n");
    if (newline >= 0) {
      const joined = next.slice(0, newline) + next.slice(newline + 1);
      const withText = replaceAt(blocks, index, { ...block, text: joined });
      const split = splitAt(withText, index, newline);
      onChangeBlocks(split.blocks);
      request(split.focus, 0);
      return;
    }

    const retyped = applyMarkdownPrefix(block, next);
    if (retyped) {
      onChangeBlocks(replaceAt(blocks, index, retyped));
      request(index, retyped.text.length);
      return;
    }

    onChangeBlocks(replaceAt(blocks, index, { ...block, text: next }));
  };

  const handleBackspaceAtStart = (index: number) => {
    const merged = mergeBackwards(blocks, index);
    onChangeBlocks(merged.blocks);
    request(merged.focus, merged.offset);
  };

  const handleInsert = (index: number) => {
    const next = insertAfter(blocks, index);
    onChangeBlocks(next.blocks);
    request(next.focus, 0);
  };

  const handleMove = (index: number, delta: number) => {
    const next = moveBlock(blocks, index, delta);
    if (next.focus === index) return;
    onChangeBlocks(next.blocks);
    onFocusedChange(next.focus);
  };

  const lane = editable && size === "roomy";

  return (
    <View className={`gap-0.5 ${lane ? "pl-13" : ""}`}>
      <TextInput
        value={title}
        onChangeText={onChangeTitle}
        editable={editable}
        placeholder={strings.note.titlePlaceholder}
        placeholderTextColor={organic.neutral[600]}
        multiline
        accessibilityLabel={strings.note.titlePlaceholder}
        className={`font-cap text-fg ${size === "roomy" ? "text-[33px] leading-[40px]" : "text-[26px] leading-[32px]"}`}
        style={{ letterSpacing: -0.6 }}
        onFocus={() => onFocusedChange(NO_BLOCK)}
      />

      <View className="h-[10px]" />

      {ensureBlocks(blocks).map((block, index, rows) => (
        <BlockRow
          key={block.id}
          block={block}
          index={index}
          editable={editable}
          isFocused={focused === index}
          pending={pending && pending.index === index ? pending : null}
          onFocus={() => onFocusedChange(index)}
          onChangeText={(next) => handleText(index, next)}
          onBackspaceAtStart={() => handleBackspaceAtStart(index)}
          onToggle={() => onChangeBlocks(toggleChecked(blocks, index))}
          ordinal={ordinalOf(blocks, index)}
          gutter={lane}
          onInsert={() => handleInsert(index)}
          onMoveUp={index > 0 ? () => handleMove(index, -1) : undefined}
          onMoveDown={index < rows.length - 1 ? () => handleMove(index, 1) : undefined}
        />
      ))}
    </View>
  );
}

function ordinalOf(blocks: readonly Block[], index: number): number {
  let n = 1;
  for (let i = index - 1; i >= 0; i--) {
    if (blocks[i]?.type !== BlockType.NUMBERED) break;
    n += 1;
  }
  return n;
}

interface BlockRowProps {
  block: Block;
  index: number;
  editable: boolean;
  isFocused: boolean;
  pending: PendingFocus | null;
  ordinal: number;
  gutter: boolean;
  onFocus: () => void;
  onChangeText: (next: string) => void;
  onBackspaceAtStart: () => void;
  onToggle: () => void;
  onInsert: () => void;
  onMoveUp?: () => void;
  onMoveDown?: () => void;
}

function BlockRow({
  block,
  editable,
  pending,
  ordinal,
  gutter,
  onFocus,
  onChangeText,
  onBackspaceAtStart,
  onToggle,
  onInsert,
  onMoveUp,
  onMoveDown,
}: BlockRowProps) {
  const input = useRef<TextInput>(null);
  const caret = useRef({ start: 0, end: 0 });
  const [selection, setSelection] = useState<{ start: number; end: number } | undefined>(undefined);

  useEffect(() => {
    if (!pending) return;
    input.current?.focus();
    setSelection({ start: pending.offset, end: pending.offset });
    const id = setTimeout(() => setSelection(undefined), 0);
    return () => clearTimeout(id);
  }, [pending]);

  const wrap = (body: ReactNode) => (
    <View>
      {gutter ? (
        <BlockGutter onInsert={onInsert} onMoveUp={onMoveUp} onMoveDown={onMoveDown} />
      ) : null}
      {body}
    </View>
  );

  if (block.type === BlockType.DIVIDER) {
    return wrap(
      <View className="py-3.5">
        <Divider />
      </View>,
    );
  }

  if (block.type === BlockType.IMAGE) {
    return wrap(
      <View className="py-1.5">
        <View
          accessible
          accessibilityLabel={strings.note.imageBlock}
          className="h-[92px] flex-row items-center justify-center gap-sm rounded-xl border border-dashed border-neutral-300"
        >
          <Icon name="image" size={16} color={organic.neutral[600]} />
          <Text className="font-fig text-[12px] text-neutral-700">{strings.note.imageBlock}</Text>
        </View>
      </View>,
    );
  }

  const text = (
    <TextInput
      ref={input}
      value={block.text}
      onChangeText={onChangeText}
      editable={editable}
      multiline
      scrollEnabled={false}
      selection={selection}
      onSelectionChange={(e) => {
        caret.current = e.nativeEvent.selection;
      }}
      onFocus={onFocus}
      onKeyPress={(e) => {
        if (e.nativeEvent.key !== "Backspace") return;
        if (caret.current.start === 0 && caret.current.end === 0) onBackspaceAtStart();
      }}
      placeholder={block.text === "" ? strings.note.placeholder : undefined}
      placeholderTextColor={organic.neutral[600]}
      accessibilityLabel={strings.note.placeholder}
      className={`flex-1 ${textClassFor(block.type)} ${
        block.type === BlockType.TODO && block.checked ? "line-through opacity-50" : ""
      }`}
      style={headingStyle(block)}
    />
  );

  if (block.type === BlockType.TODO) {
    return wrap(
      <View className="flex-row items-start gap-2.5 py-0.75">
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: block.checked }}
          accessibilityLabel={block.text || strings.blockBar.todo}
          disabled={!editable}
          onPress={onToggle}
          hitSlop={8}
          className="mt-1.5 h-[17px] w-[17px] flex-none items-center justify-center rounded-md border"
          style={{
            borderColor: block.checked ? organic.accent.DEFAULT : organic.neutral[600],
            backgroundColor: block.checked ? organic.accent.DEFAULT : "transparent",
            borderWidth: 1.5,
          }}
        >
          {block.checked ? <Icon name="check" size={11} color={organic.bg} /> : null}
        </Pressable>
        {text}
      </View>,
    );
  }

  if (block.type === BlockType.BULLET) {
    return wrap(
      <View className="flex-row items-start gap-2.5 py-0.5">
        <View className="mt-2.75 h-1.25 w-1.25 flex-none rounded-full bg-neutral-600" />
        {text}
      </View>,
    );
  }

  if (block.type === BlockType.NUMBERED) {
    return wrap(
      <View className="flex-row items-start gap-2.5 py-0.5">
        <Text className="mt-xs w-[16px] flex-none text-right font-fig-med text-[14px] text-neutral-700">
          {ordinal}.
        </Text>
        {text}
      </View>,
    );
  }

  if (block.type === BlockType.QUOTE) {
    return wrap(
      <View
        className="my-1.5 flex-row pl-3.5"
        style={{ borderLeftWidth: 2, borderLeftColor: organic.accent.DEFAULT }}
      >
        {text}
      </View>,
    );
  }

  if (block.type === BlockType.CODE) {
    return wrap(<View className="my-1.5 rounded-xl bg-surface px-3 py-2.25">{text}</View>);
  }

  return wrap(<View className="flex-row py-0.5">{text}</View>);
}

function BlockGutter({
  onInsert,
  onMoveUp,
  onMoveDown,
}: {
  onInsert: () => void;
  onMoveUp?: () => void;
  onMoveDown?: () => void;
}) {
  return (
    <View className="absolute left-[-52px] top-[4px] flex-row items-center gap-0.25">
      <GutterButton icon="plus" label={strings.gutter.insert} onPress={onInsert} />
      <GutterButton icon="caret-up" label={strings.gutter.moveUp} onPress={onMoveUp} />
      <GutterButton icon="caret-down" label={strings.gutter.moveDown} onPress={onMoveDown} />
    </View>
  );
}

function GutterButton({
  icon,
  label,
  onPress,
}: {
  icon: IconName;
  label: string;
  onPress?: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !onPress }}
      disabled={!onPress}
      onPress={onPress}
      hitSlop={6}
      className={`p-0.25 ${onPress ? "" : "opacity-30"}`}
    >
      <Icon name={icon} size={13} color={organic.neutral[600]} />
    </Pressable>
  );
}

const PARAGRAPH_CLASS = "font-fig text-[15.5px] leading-[27px] text-fg";

const TEXT_CLASS: Partial<Record<BlockType, string>> = {
  [BlockType.PARAGRAPH]: PARAGRAPH_CLASS,
  [BlockType.HEADING]: "font-fig-semi text-fg",
  [BlockType.TODO]: "font-fig text-[15.5px] leading-[24px] text-fg",
  [BlockType.BULLET]: "font-fig text-[15.5px] leading-[26px] text-fg",
  [BlockType.NUMBERED]: "font-fig text-[15.5px] leading-[26px] text-fg",
  [BlockType.QUOTE]: "font-fig text-[15px] leading-[25px] text-neutral-900",
  [BlockType.CODE]: "font-fig text-[13.5px] leading-[21px] text-accent-800",
};

function textClassFor(type: BlockType): string {
  return TEXT_CLASS[type] ?? PARAGRAPH_CLASS;
}

const HEADING_SIZES: Record<number, { fontSize: number; lineHeight: number }> = {
  1: { fontSize: 24, lineHeight: 32 },
  2: { fontSize: 20, lineHeight: 28 },
  3: { fontSize: 17, lineHeight: 25 },
};

const HEADING_DEFAULT = { fontSize: 20, lineHeight: 28 };

function headingStyle(block: Block) {
  if (block.type !== BlockType.HEADING) return undefined;
  const size = HEADING_SIZES[block.level] ?? HEADING_DEFAULT;
  return { fontSize: size.fontSize, lineHeight: size.lineHeight, letterSpacing: -0.3 };
}
