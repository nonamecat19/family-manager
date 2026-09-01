import { BlockType } from "@fm/sdk/notes/v1/notes_pb";
import { Pressable, Text, View } from "react-native";

import { strings } from "../i18n/index.ts";
import { Icon, nocturne, type IconName } from "../nocturne/index.ts";

/**
 * The block bar. One control, two placements:
 *  - `floating` is the desktop pill the design parks at the bottom of the editor column;
 *  - `docked` is the mobile bar pinned above the keyboard, with "Done" on the right.
 *
 * It retypes the FOCUSED block, and carries a button for every BlockType the product exposes.
 * When no block is focused — a note just opened, or the caret in the title — and when the
 * focused index has come to rest on a block that holds no text, there is nothing to retype:
 * the caller passes `canRetype={false}` and the type buttons go dim rather than acting on a
 * block the user never put a caret in. Comment and Done are not retypes and stay live.
 *
 * Four glyphs the design draws are deliberately absent.
 *  - bold and italic (1c): the proto's Block has no marks, so a B and an I would be two
 *    buttons that cannot do anything;
 *  - the paperclip and the microphone (1e): there is no ATTACHMENT or VOICE BlockType and no
 *    rpc that would store either, so they would be buttons with nowhere to put their bytes.
 *    They come back the day the contract grows them, not before.
 *  - the image/camera button (1c and 1e): IMAGE BLOCKS ARE DEFERRED PAST V1, and this is not
 *    an oversight. libs/go/storage puts an anonymous-read policy on every bucket it creates,
 *    so a photo inside a never-shared note would be fetchable by anyone holding its URL and
 *    revoking the share would not revoke the image — the opposite of "private by default",
 *    which is the product. BLOCK_TYPE_IMAGE and UploadNoteImage stay in the contract; nothing
 *    in this app offers them, and the notes service never provisions the bucket. The button
 *    comes back when object storage can hold a private object, not before. A note that
 *    already contains an image block still renders one — see BlockEditor.
 */

export interface BlockBarProps {
  active: BlockType;
  onSetType: (type: BlockType) => void;
  onComment?: () => void;
  onDone?: () => void;
  variant?: "floating" | "docked";
  /** False when nothing focused can be retyped; dims the type buttons. Default true. */
  canRetype?: boolean;
}

interface BarItem {
  type: BlockType;
  icon: IconName;
  label: string;
}

const ITEMS: readonly BarItem[] = [
  { type: BlockType.PARAGRAPH, icon: "text-aa", label: strings.blockBar.text },
  { type: BlockType.HEADING, icon: "text-h-two", label: strings.blockBar.heading },
  { type: BlockType.TODO, icon: "check-square", label: strings.blockBar.todo },
  { type: BlockType.BULLET, icon: "list-bullets", label: strings.blockBar.bullet },
  { type: BlockType.NUMBERED, icon: "list-numbers", label: strings.blockBar.numbered },
  { type: BlockType.QUOTE, icon: "quotes", label: strings.blockBar.quote },
  { type: BlockType.CODE, icon: "code", label: strings.blockBar.code },
  // A rule is INSERTED rather than retyped-into — the caller routes DIVIDER through
  // insertDivider — so it never reads as the active type, only as an action.
  { type: BlockType.DIVIDER, icon: "minus", label: strings.blockBar.divider },
];

export function BlockBar({
  active,
  onSetType,
  onComment,
  onDone,
  variant = "floating",
  canRetype = true,
}: BlockBarProps) {
  const floating = variant === "floating";
  const glyph = floating ? 16 : 19;

  return (
    <View
      className={
        floating
          ? "flex-row items-center gap-[2px] self-center rounded-lg bg-surface px-[7px] py-[5px]"
          : "flex-row items-center gap-[2px] border-t border-neutral-800 bg-surface px-[10px] py-[7px]"
      }
    >
      {ITEMS.map((item) => (
        <BarButton
          key={item.type}
          icon={item.icon}
          label={item.label}
          size={glyph}
          disabled={!canRetype}
          active={active === item.type}
          onPress={() => onSetType(item.type)}
        />
      ))}

      {onComment ? (
        <>
          <Separator visible={floating} />
          <BarButton
            icon="chat-teardrop-text"
            label={strings.blockBar.comment}
            size={glyph}
            // A comment is note-level, so it has nothing to do with the focused block.
            disabled={false}
            active={false}
            onPress={onComment}
          />
        </>
      ) : null}

      {onDone ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={strings.note.done}
          onPress={onDone}
          className="ml-auto px-[6px] py-[4px]"
        >
          <Text className="font-med text-[14px] text-accent">{strings.note.done}</Text>
        </Pressable>
      ) : null}
    </View>
  );
}

function Separator({ visible }: { visible: boolean }) {
  if (!visible) return null;
  return <View className="mx-[4px] h-[16px] w-[1px] bg-neutral-800" />;
}

function BarButton({
  icon,
  label,
  size,
  active,
  disabled,
  onPress,
}: {
  icon: IconName;
  label: string;
  size: number;
  active: boolean;
  disabled: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected: active, disabled }}
      disabled={disabled}
      onPress={onPress}
      hitSlop={4}
      className={`items-center justify-center rounded-sm p-[5px] ${disabled ? "opacity-40" : ""}`}
    >
      <Icon
        name={icon}
        size={size}
        color={active ? nocturne.accent.DEFAULT : nocturne.neutral[400]}
        weight={active && icon === "check-square" ? "fill" : "regular"}
      />
    </Pressable>
  );
}
