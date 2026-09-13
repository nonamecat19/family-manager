import { BlockType } from "@fm/sdk/notes/v1/notes_pb";
import { Pressable, Text, View } from "react-native";

import { strings } from "../i18n/index.ts";
import { Icon, nocturne, type IconName } from "../nocturne/index.ts";


export interface BlockBarProps {
  active: BlockType;
  onSetType: (type: BlockType) => void;
  onComment?: () => void;
  onDone?: () => void;
  variant?: "floating" | "docked";
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
