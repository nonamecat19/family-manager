import { BlockType, type Notebook } from "@fm/sdk/notes/v1/notes_pb";
import { useState } from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { makeBlock } from "../editor/index.ts";
import { strings } from "../i18n/index.ts";
import { offlineCopy } from "./copy.ts";
import { Chip, Icon, IconButton, PrimaryButton, nocturne } from "../nocturne/index.ts";
import { useCaptureQueue } from "./queue.ts";


export interface CaptureSheetProps {
  visible: boolean;
  onClose: () => void;
  notebooks: readonly Notebook[];
  defaultNotebookId?: string;
}

export function CaptureSheet({ visible, onClose, notebooks, defaultNotebookId = "" }: CaptureSheetProps) {
  const queue = useCaptureQueue();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [asTask, setAsTask] = useState(false);
  const [notebookId, setNotebookId] = useState(defaultNotebookId);
  const [failed, setFailed] = useState(false);

  const reset = () => {
    setTitle("");
    setBody("");
    setAsTask(false);
    setFailed(false);
  };

  const save = async () => {
    const trimmedTitle = title.trim();
    const trimmedBody = body.trim();
    if (trimmedTitle === "" && trimmedBody === "") return;
    try {
      await queue.capture({
        notebookId,
        title: trimmedTitle === "" ? firstLine(trimmedBody) : trimmedTitle,
        blocks: [
          makeBlock({
            type: asTask ? BlockType.TODO : BlockType.PARAGRAPH,
            text: trimmedBody,
          }),
        ],
      });
    } catch (error) {
      console.warn("[notes] the capture was not saved", error);
      setFailed(true);
      return;
    }
    reset();
    onClose();
  };

  const notebook = notebooks.find((n) => n.id === notebookId);

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.common.close}
        onPress={onClose}
        className="flex-1"
        style={SCRIM}
      />
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <View
          className="gap-[14px] bg-surface px-[18px] pb-[22px] pt-[10px]"
          style={{ borderTopLeftRadius: 20, borderTopRightRadius: 20 }}
        >
          <View className="h-[4px] w-[36px] self-center rounded-sm bg-neutral-800" />

          <View className="flex-row items-center gap-[8px]">
            <Text className="font-med text-[15px] text-fg">{strings.capture.title}</Text>
            <View
              className="flex-row items-center gap-[5px] rounded-lg px-[8px] py-[3px]"
              style={{ backgroundColor: queue.online ? nocturne.accent[900] : nocturne.accent[800] }}
            >
              <Icon
                name={queue.online ? "cloud-check" : "cloud-slash"}
                size={12}
                color={nocturne.accent[200]}
              />
              <Text className="font-sans text-[11px] text-accent-200">
                {queue.online ? strings.capture.online : strings.capture.offline}
              </Text>
            </View>
            <View className="ml-auto">
              <IconButton icon="x" label={strings.common.close} onPress={onClose} size={17} color={nocturne.neutral[500]} />
            </View>
          </View>

          <View className="gap-[6px]">
            <TextInput
              value={title}
              onChangeText={setTitle}
              placeholder={strings.capture.titlePlaceholder}
              placeholderTextColor={nocturne.neutral[600]}
              accessibilityLabel={strings.capture.titlePlaceholder}
              className="font-med text-[19px] text-fg"
            />
            <TextInput
              value={body}
              onChangeText={setBody}
              placeholder={strings.capture.bodyPlaceholder}
              placeholderTextColor={nocturne.neutral[600]}
              accessibilityLabel={strings.capture.bodyPlaceholder}
              multiline
              className="min-h-[64px] font-sans text-[15.5px] leading-[25px] text-fg"
            />
          </View>

          <View className="flex-row gap-[7px]">
            <Chip
              label={strings.capture.kindNote}
              icon="file-text"
              active={!asTask}
              onPress={() => setAsTask(false)}
            />
            <Chip
              label={strings.capture.kindTask}
              icon="check-square"
              active={asTask}
              onPress={() => setAsTask(true)}
            />
          </View>

          {notebooks.length > 0 ? (
            <ScrollView horizontal showsHorizontalScrollIndicator={false} className="max-h-[40px]">
              <View className="flex-row gap-[7px]">
                <Chip
                  label={strings.capture.noNotebook}
                  tone="neutral"
                  active={notebookId === ""}
                  onPress={() => setNotebookId("")}
                />
                {notebooks.map((n) => (
                  <Chip
                    key={n.id}
                    label={n.name}
                    icon="folder-simple"
                    tone="accent2"
                    active={notebookId === n.id}
                    onPress={() => setNotebookId(n.id)}
                  />
                ))}
              </View>
            </ScrollView>
          ) : null}

          {failed ? (
            <Text className="font-sans text-[12px] text-accent2-300">
              {offlineCopy.captureNotSaved}
            </Text>
          ) : null}

          <View className="flex-row items-center gap-[10px]">
            <Text className="font-sans text-[12px] text-neutral-500">
              {queue.queued.length > 0
                ? strings.capture.queued(queue.queued.length)
                : notebook
                  ? notebook.name
                  : strings.capture.noNotebook}
            </Text>
            <View className="ml-auto">
              <PrimaryButton
                title={strings.capture.save}
                onPress={() => void save()}
                disabled={title.trim() === "" && body.trim() === ""}
              />
            </View>
          </View>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

function firstLine(text: string): string {
  const line = text.split("\n")[0] ?? "";
  return line.length > 60 ? `${line.slice(0, 57)}…` : line || strings.common.untitled;
}

const SCRIM = { backgroundColor: nocturne.bg, opacity: 0.62 } as const;
