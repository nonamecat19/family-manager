import { BlockType, type Notebook } from "@fm/sdk/notes/v1/notes_pb";
import { useState } from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { makeBlock } from "../editor/index.ts";
import { strings } from "../i18n/index.ts";
import { offlineCopy } from "./copy.ts";
import { Chip, Icon, IconButton, PrimaryButton, nocturne } from "../nocturne/index.ts";
import { useCaptureQueue } from "./queue.ts";

/**
 * Quick capture (artboard 1f): the sheet the mobile FAB opens.
 *
 * It never calls CreateNote directly. Everything goes through the queue, which writes to disk
 * first and sends when there is a network — so the sheet behaves the same on the train as it
 * does at home, and the pill tells the truth about which one is happening.
 *
 * The design's Photo chip (1f) is gone and is not coming back in v1 — not an oversight, a
 * decision. Image blocks are deferred: libs/go/storage puts an anonymous-read policy on every
 * bucket it creates, so a photo captured into a private note would be fetchable by anyone
 * holding its URL and un-sharing the note would not revoke it. Until object storage can hold a
 * private object there is nowhere safe to put the bytes, so the sheet does not offer to take
 * them. (The chip had a second problem even then: UploadNoteImage needs a note_id, and a note
 * captured offline does not have one yet.)
 *
 * The Voice chip is absent for the older reason: there is no VOICE BlockType and no rpc that
 * would store the audio, so it would be a chip with nowhere to put its bytes.
 *
 * The notebook picker offers every notebook ListNotebooks returned, and some of those are shared
 * with the caller VIEW-only — capturing into one is refused server-side, after the note is
 * written. The sheet CANNOT filter them out today: Notebook carries owner_user_id but no
 * can_edit (notes.proto:176), and the proto is explicit that permission must not be re-derived
 * on the client from owner + shares. Filtering by ownership would also hide the notebooks shared
 * for EDIT, which are exactly the ones a family shares. So the refusal is handled instead of
 * predicted: the queue keeps the note, records why the server said no, and Settings offers to
 * refile it under the user's own notes. The proper fix is a `can_edit` on Notebook — a contract
 * change, reported rather than made here.
 */

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
      // The queue rejects when the note did not reach the disk. Keep the sheet open with the
      // user's text still in it: clearing a draft that was saved nowhere is how a capture app
      // loses a note. The queue has already logged why.
      console.warn("[notes] the capture was not saved", error);
      // Silence here read as "saved" to the user, because the sheet simply did not move. It
      // says so instead — the queue refuses to write a note it cannot attribute to the
      // signed-in user, and that is a real (if brief) state right after a sign-in.
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

/**
 * The overlay the design dims the app with behind a sheet. Drawn as the ground colour at
 * opacity rather than as a fourth hard-coded rgba: the ground is a token, and the scrim is
 * "the ground, mostly opaque".
 */
const SCRIM = { backgroundColor: nocturne.bg, opacity: 0.62 } as const;
