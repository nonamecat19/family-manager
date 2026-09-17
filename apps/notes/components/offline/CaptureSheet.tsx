import { BlockType, type Notebook } from "@fm/sdk/notes/v1/notes_pb";
import { useState } from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { makeBlock } from "../editor/index.ts";
import { strings } from "../i18n/index.ts";
import { offlineCopy } from "./copy.ts";
import { Chip, Icon, IconButton, PrimaryButton, organic } from "@fm/ui";
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
          className="gap-3.5 bg-surface px-4.5 pb-5.5 pt-2.5"
          style={{ borderTopLeftRadius: 20, borderTopRightRadius: 20 }}
        >
          <View className="h-[4px] w-[36px] self-center rounded-md bg-neutral-400" />

          <View className="flex-row items-center gap-sm">
            <Text className="font-fig-med text-15 text-fg">{strings.capture.title}</Text>
            <View
              className="flex-row items-center gap-1.25 rounded-2xl px-sm py-0.75"
              style={{ backgroundColor: queue.online ? organic.accent[100] : organic.accent[200] }}
            >
              <Icon
                name={queue.online ? "cloud-check" : "cloud-slash"}
                size={12}
                color={organic.accent[800]}
              />
              <Text className="font-fig text-11 text-accent-800">
                {queue.online ? strings.capture.online : strings.capture.offline}
              </Text>
            </View>
            <View className="ml-auto">
              <IconButton icon="x" label={strings.common.close} onPress={onClose} size={17} color={organic.neutral[700]} />
            </View>
          </View>

          <View className="gap-1.5">
            <TextInput
              value={title}
              onChangeText={setTitle}
              placeholder={strings.capture.titlePlaceholder}
              placeholderTextColor={organic.neutral[600]}
              accessibilityLabel={strings.capture.titlePlaceholder}
              className="font-fig-med text-19 text-fg"
            />
            <TextInput
              value={body}
              onChangeText={setBody}
              placeholder={strings.capture.bodyPlaceholder}
              placeholderTextColor={organic.neutral[600]}
              accessibilityLabel={strings.capture.bodyPlaceholder}
              multiline
              className="min-h-[64px] font-fig text-15.5 leading-[25px] text-fg"
            />
          </View>

          <View className="flex-row gap-1.75">
            <Chip
              label={strings.capture.kindNote}
              active={!asTask}
              onPress={() => setAsTask(false)}
            />
            <Chip
              label={strings.capture.kindTask}
              active={asTask}
              onPress={() => setAsTask(true)}
            />
          </View>

          {notebooks.length > 0 ? (
            <ScrollView horizontal showsHorizontalScrollIndicator={false} className="max-h-[40px]">
              <View className="flex-row gap-1.75">
                <Chip
                  label={strings.capture.noNotebook}
                  active={notebookId === ""}
                  onPress={() => setNotebookId("")}
                />
                {notebooks.map((n) => (
                  <Chip
                    key={n.id}
                    label={n.name}
                    tone="accent2"
                    active={notebookId === n.id}
                    onPress={() => setNotebookId(n.id)}
                  />
                ))}
              </View>
            </ScrollView>
          ) : null}

          {failed ? (
            <Text className="font-fig text-12 text-accent2-800">
              {offlineCopy.captureNotSaved}
            </Text>
          ) : null}

          <View className="flex-row items-center gap-2.5">
            <Text className="font-fig text-12 text-neutral-700">
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

const SCRIM = { backgroundColor: organic.bg, opacity: 0.62 } as const;
