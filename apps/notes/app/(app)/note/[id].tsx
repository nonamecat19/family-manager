import {
  useActivity,
  useAddNoteComment,
  useArchiveNote,
  useDeleteNote,
  useMoveNote,
  useNote,
  useNoteComments,
  useResolveComment,
  useToggleStar,
} from "@fm/api";
import {
  ActivityKind,
  BlockType,
  SharePermission,
  ShareSubject,
  type Activity,
  type Block,
  type Comment,
  type Note,
} from "@fm/sdk/notes/v1/notes_pb";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from "react-native";

import {
  BLOCK_COLUMN_WIDTH,
  BlockBar,
  BlockEditor,
  GUTTER_LANE,
  NO_BLOCK,
  canRetype,
  insertDivider,
  outline,
  setType,
  useNoteDraft,
} from "../../../components/editor/index.ts";
import { longDate, relative, strings } from "../../../components/i18n/index.ts";
import {
  Avatar,
  AvatarStack,
  Divider,
  EmptyState,
  Icon,
  IconButton,
  PrimaryButton,
  Screen,
  nocturne,
} from "../../../components/nocturne/index.ts";
import { ShareSheet, useMemberDirectory } from "../../../components/share.tsx";
import { ActionSheet, PANE, firstName, usePaneFit, useShell, type SheetAction } from "../_layout.tsx";

/**
 * The note editor — artboard 1c on desktop, 1e on mobile.
 *
 * DELIBERATE DEPARTURE FROM THE MOCK. The design draws live presence ("3 editing now"), a
 * typing indicator, and comments pinned to individual blocks. There is no realtime channel in
 * this system — no CRDT, no presence, no websocket — so rather than fake it, the same chrome
 * is rendered from data that is actually true:
 *
 *   - the avatar stack is the note's SHARES, i.e. who can see this note;
 *   - "Maya edited 4m ago" is last_edited_by_user_id + updated_at;
 *   - the margin comments are ListComments, which are note-level in the proto, so they sit
 *     beside the block column rather than pointing at one block;
 *   - the typing indicator is gone entirely. Nothing could make it true.
 *
 * Concurrent editing is handled by expected_version instead: the conflict banner is the
 * honest version of "3 editing now".
 *
 * IMAGE BLOCKS ARE NOT IN V1, and this screen is where they used to be made. There is no
 * picker, no permission request and no UploadNoteImage call here on purpose: libs/go/storage
 * sets an anonymous-read policy on every bucket it creates, so a photo in a note that was
 * never shared would be readable by anyone holding the URL, and un-sharing the note would not
 * take the image back. "Private by default" is the product, so the feature waits for storage
 * that can keep a private object. The rpc and BLOCK_TYPE_IMAGE stay in the contract, unused;
 * an image block that arrives from an older client still renders (BlockEditor draws a labelled
 * placeholder) and is written back untouched by the save loop.
 */
export default function NoteScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const noteId = id ?? "";
  // Which of 1c's columns this window can afford. The margin comments and the right rail are
  // the two that give way; see usePaneFit in the shell for the arithmetic.
  const panes = usePaneFit();
  const desktop = panes.desktop;
  const router = useRouter();
  const shell = useShell();

  const query = useNote(noteId);
  const note = query.data ?? null;
  const canEdit = note?.canEdit ?? false;
  const draft = useNoteDraft(note, canEdit);

  // NO_BLOCK, not 0. Until the user puts a caret somewhere there is no focused block, and 0 is
  // a real one: starting there made the first press on the block bar retype the note's first
  // block. An IMAGE block renders no TextInput, so it never reports focus and never corrects
  // that guess — opening a note that starts with an image and pressing any type button
  // replaced the image with an empty paragraph.
  const [focused, setFocused] = useState(NO_BLOCK);
  const [shareOpen, setShareOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [moveOpen, setMoveOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  // The desktop comment button opens the composer the margin column draws. Comments are
  // note-level here (the proto has no block anchor), so it is one thread, not one per block.
  const [composing, setComposing] = useState(false);
  const closeComposer = useCallback(() => setComposing(false), []);
  const toggleStar = useToggleStar();
  const archiveNote = useArchiveNote();
  const moveNote = useMoveNote();
  const deleteNote = useDeleteNote();
  const { nameOf } = useMemberDirectory();

  const applyType = (type: BlockType) => {
    // Both halves of "is there something to retype": an index that points at a block at all,
    // and a block that holds text. The gutter's move can leave the index on an image or a
    // rule, so the index alone is never the answer. setType refuses the same cases.
    if (!canRetype(draft.blocks, focused)) return;
    if (type === BlockType.DIVIDER) {
      const next = insertDivider(draft.blocks, focused);
      draft.setBlocks(next.blocks);
      setFocused(next.focus);
      return;
    }
    draft.setBlocks(setType(draft.blocks, focused, type));
  };

  if (query.isPending) {
    return (
      <Screen>
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator color={nocturne.accent.DEFAULT} />
        </View>
      </Screen>
    );
  }

  if (!note) {
    return (
      <Screen>
        <EmptyState
          title={strings.note.notFound}
          body={strings.note.notFoundBody}
          icon="file-text"
          action={{ label: strings.note.back, onPress: () => router.back() }}
        />
      </Screen>
    );
  }

  // What the bar lights up, and whether it is live at all. UNSPECIFIED matches no button, so
  // with nothing focused the bar shows no type as active instead of claiming the note starts
  // as a paragraph.
  const retypeable = canRetype(draft.blocks, focused);
  const activeType = retypeable
    ? (draft.blocks[focused]?.type ?? BlockType.PARAGRAPH)
    : BlockType.UNSPECIFIED;
  const shareNames = shareNamesOf(note, nameOf);
  const editorName = nameOf(note.lastEditedByUserId);

  const conflict =
    draft.state === "conflict" ? (
      <ConflictBanner
        onReload={() => void query.refetch().then((result) => draft.reload(result.data ?? undefined))}
        onOverwrite={draft.overwrite}
      />
    ) : null;

  const editor = (
    <BlockEditor
      title={draft.title}
      onChangeTitle={draft.setTitle}
      blocks={draft.blocks}
      onChangeBlocks={draft.setBlocks}
      editable={canEdit}
      focused={focused}
      onFocusedChange={setFocused}
      size={desktop ? "roomy" : "compact"}
    />
  );

  /**
   * The header's overflow menu, both layouts.
   *
   * Archive is a write, so it sits behind `note.canEdit` — the server's answer, never
   * re-derived from owner_user_id here — next to move and delete. `archived` is the value to
   * set rather than a toggle, and the screen deliberately stays put afterwards: the note leaves
   * the lists but the route it is on still resolves, which is what makes "take it back out"
   * the same menu item one press later.
   */
  const overflow: SheetAction[] = [
    {
      key: "star",
      label: note.starred ? strings.note.starOff : strings.note.starOn,
      icon: "star",
      onPress: () => toggleStar.mutate({ noteId: note.id, starred: !note.starred }),
    },
    ...(canEdit
      ? [
          {
            key: "archive",
            label: note.archived ? strings.rail.restoreNotebook : strings.rail.archive,
            icon: "archive" as const,
            onPress: () => archiveNote.mutate({ noteId: note.id, archived: !note.archived }),
          },
          {
            key: "move",
            label: strings.note.moveTo,
            icon: "folder-simple" as const,
            onPress: () => setMoveOpen(true),
          },
          {
            key: "delete",
            label: strings.note.delete,
            icon: "trash" as const,
            tone: "danger" as const,
            onPress: () => setDeleteOpen(true),
          },
        ]
      : []),
  ];

  const notice = deleteNote.isError ? (
    <Text className="px-[22px] py-[8px] font-sans text-[12px] text-neutral-500">
      {strings.note.deleteFailed}
    </Text>
  ) : null;

  const sheet = (
    <>
      <ShareSheet
        visible={shareOpen}
        onClose={() => setShareOpen(false)}
        noteId={note.id}
        ownerUserId={note.ownerUserId}
      />

      <ActionSheet
        visible={menuOpen}
        title={note.title || strings.common.untitled}
        onClose={() => setMenuOpen(false)}
        actions={overflow}
      />

      <ActionSheet
        visible={moveOpen}
        title={strings.note.moveTo}
        onClose={() => setMoveOpen(false)}
        actions={[
          {
            key: "none",
            label: strings.note.noNotebook,
            selected: note.notebookId === "",
            onPress: () => moveNote.mutate({ noteId: note.id, notebookId: "" }),
          },
          ...shell.notebooks.map((notebook) => ({
            key: notebook.id,
            label: notebook.name,
            icon: "folder-simple" as const,
            selected: notebook.id === note.notebookId,
            onPress: () => moveNote.mutate({ noteId: note.id, notebookId: notebook.id }),
          })),
        ]}
      />

      <ActionSheet
        visible={deleteOpen}
        title={strings.note.deleteConfirm}
        onClose={() => setDeleteOpen(false)}
        actions={[
          {
            key: "confirm",
            label: strings.note.delete,
            icon: "trash",
            tone: "danger",
            onPress: () =>
              deleteNote.mutate(note.id, {
                onSuccess: () => router.replace("/(app)"),
              }),
          },
        ]}
      />
    </>
  );

  if (desktop) {
    return (
      <View className="flex-1 bg-bg">
        <View className="flex-row items-center gap-[12px] border-b border-neutral-900 px-[22px] py-[12px]">
          <IconButton
            icon="arrow-left"
            label={strings.note.back}
            size={16}
            color={nocturne.neutral[500]}
            onPress={() => router.back()}
          />
          <Text className="font-sans text-[12.5px] text-neutral-500">
            {breadcrumb(shell.notebooks.find((n) => n.id === note.notebookId)?.name)}
          </Text>
          <Text numberOfLines={1} className="shrink font-sans text-[12.5px] text-fg">
            {note.title || strings.common.untitled}
          </Text>

          <View className="ml-auto flex-row items-center gap-[14px]">
            {shareNames.length > 0 ? (
              <View
                className="flex-row items-center gap-[7px] rounded-lg py-[4px] pl-[5px] pr-[10px]"
                style={{ backgroundColor: nocturne.accent[900] }}
              >
                <AvatarStack names={shareNames} size={24} max={3} />
                <Text className="font-sans text-[11.5px] text-accent-200">
                  {editorName !== ""
                    ? strings.note.editedBy(firstName(editorName), relative(note.updatedAt))
                    : strings.note.updated(relative(note.updatedAt))}
                </Text>
              </View>
            ) : null}

            <SaveState state={draft.state} />

            <IconButton
              icon="star"
              label={note.starred ? strings.note.starOff : strings.note.starOn}
              size={16}
              weight={note.starred ? "fill" : "regular"}
              color={note.starred ? nocturne.accent.DEFAULT : nocturne.neutral[500]}
              onPress={() => toggleStar.mutate({ noteId: note.id, starred: !note.starred })}
            />

            <Pressable
              accessibilityRole="button"
              accessibilityLabel={strings.note.share}
              onPress={() => setShareOpen(true)}
              className="h-[30px] flex-row items-center gap-[6px] rounded-md border border-accent px-[12px]"
            >
              <Icon name="user-plus" size={14} color={nocturne.accent.DEFAULT} />
              <Text className="font-med text-[13px] text-accent">{strings.note.share}</Text>
            </Pressable>

            <IconButton
              icon="dots-three"
              label={strings.note.more}
              size={18}
              color={nocturne.neutral[500]}
              onPress={() => setMenuOpen(true)}
            />
          </View>
        </View>

        {conflict}
        {notice}

        <View className="min-h-0 flex-1 flex-row">
          <View className="min-w-0 flex-1">
            <ScrollView
              // The floating block bar hangs over the bottom of this column, so an editable
              // note reserves the room it covers rather than hiding the last thing written.
              contentContainerClassName={`px-[40px] pt-[44px] ${canEdit ? "pb-[96px]" : "pb-[44px]"}`}
            >
              {/* The design's measure, plus the lane the block gutter is drawn in when the
                  note is editable — so the text is 560px wide either way. */}
              <View
                className="w-full self-center"
                style={{ maxWidth: BLOCK_COLUMN_WIDTH + (canEdit ? GUTTER_LANE : 0) }}
              >
                <NoteMeta note={note} />
                {editor}
                {!canEdit ? (
                  <Text className="pt-[18px] font-sans text-[12px] text-neutral-600">
                    {strings.note.readOnly}
                  </Text>
                ) : null}
                {panes.comments ? null : (
                  <StackedComments
                    noteId={note.id}
                    nameOf={nameOf}
                    composing={composing}
                    onComposerShown={closeComposer}
                  />
                )}
              </View>
            </ScrollView>

            {canEdit ? (
              <View className="absolute bottom-[26px] w-full items-center">
                <BlockBar
                  active={activeType}
                  canRetype={retypeable}
                  onSetType={applyType}
                  onComment={() => setComposing(true)}
                />
              </View>
            ) : null}
          </View>

          {panes.comments ? (
            <CommentsColumn
              noteId={note.id}
              nameOf={nameOf}
              composing={composing}
              onCloseComposer={closeComposer}
            />
          ) : null}
          {panes.editorRail ? (
            <EditorRail
              note={note}
              blocks={draft.blocks}
              nameOf={nameOf}
              onInvite={() => setShareOpen(true)}
            />
          ) : null}
        </View>

        {sheet}
      </View>
    );
  }

  return (
    <Screen>
      <KeyboardAvoidingView
        className="flex-1"
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        <View className="flex-row items-center gap-[10px] px-[16px] pb-[10px] pt-[6px]">
          <IconButton
            icon="caret-left"
            label={strings.note.back}
            size={20}
            color={nocturne.accent.DEFAULT}
            onPress={() => {
              draft.flush();
              router.back();
            }}
          />
          <Text className="font-sans text-[13px] text-neutral-500">
            {shell.notebooks.find((n) => n.id === note.notebookId)?.name ?? strings.list.title}
          </Text>
          <View className="ml-auto flex-row items-center gap-[12px]">
            <AvatarStack names={shareNames} size={24} max={2} />
            <IconButton
              icon="user-plus"
              label={strings.note.share}
              size={18}
              color={nocturne.neutral[400]}
              onPress={() => setShareOpen(true)}
            />
            <IconButton
              icon="dots-three"
              label={strings.note.more}
              size={18}
              color={nocturne.neutral[400]}
              onPress={() => setMenuOpen(true)}
            />
          </View>
        </View>

        {conflict}
        {notice}

        <ScrollView contentContainerClassName="px-[20px] pb-[24px]">
          <View className="flex-row items-center gap-[8px] pb-[10px]">
            <SaveState state={draft.state} />
            {editorName !== "" ? (
              <Text className="font-sans text-[11.5px] text-neutral-600">
                · {strings.note.editedBy(firstName(editorName), relative(note.updatedAt))}
              </Text>
            ) : null}
          </View>
          {editor}
          {!canEdit ? (
            <Text className="pt-[16px] font-sans text-[12px] text-neutral-600">
              {strings.note.readOnly}
            </Text>
          ) : null}
          <StackedComments noteId={note.id} nameOf={nameOf} />
        </ScrollView>

        {canEdit ? (
          <BlockBar
            variant="docked"
            active={activeType}
            canRetype={retypeable}
            onSetType={applyType}
            onDone={() => {
              draft.flush();
              router.back();
            }}
          />
        ) : null}
      </KeyboardAvoidingView>
      {sheet}
    </Screen>
  );
}

/* ------------------------------------------------------------------ pieces */

function NoteMeta({ note }: { note: Note }) {
  return (
    <View className="flex-row items-center gap-[9px] pb-[10px]">
      <Text className="font-sans text-[11.5px] text-neutral-600">
        {strings.note.created(longDate(note.createdAt))}
      </Text>
      <Text className="font-sans text-[11.5px] text-neutral-700">·</Text>
      <Text className="font-sans text-[11.5px] text-neutral-600">
        {strings.note.version(Number(note.version))}
      </Text>
    </View>
  );
}

function SaveState({ state }: { state: string }) {
  const failed = state === "error" || state === "conflict";
  const label =
    state === "saving"
      ? strings.note.saving
      : state === "dirty"
        ? strings.note.unsaved
        : failed
          ? strings.note.saveFailed
          : strings.note.saved;
  return (
    <View className="flex-row items-center gap-[6px]">
      <Icon
        name={failed ? "cloud-slash" : "cloud-check"}
        size={14}
        color={failed ? nocturne.neutral[500] : nocturne.accent[400]}
      />
      <Text className="font-sans text-[11.5px] text-neutral-500">{label}</Text>
    </View>
  );
}

/**
 * The conflict banner. The proto refuses a write whose expected_version is stale with ABORTED
 * precisely so this can exist: two people editing one shared note find out, instead of one of
 * them silently losing a paragraph. Nothing is sent again until the user picks a side.
 */
function ConflictBanner({ onReload, onOverwrite }: { onReload: () => void; onOverwrite: () => void }) {
  return (
    <View
      className="gap-[8px] border-b border-neutral-900 px-[22px] py-[12px]"
      style={{ backgroundColor: nocturne.accent[900] }}
    >
      <Text className="font-med text-[13px] text-accent-200">{strings.note.conflictTitle}</Text>
      <Text className="font-sans text-[12.5px] leading-[19px] text-neutral-400">
        {strings.note.conflictBody}
      </Text>
      <View className="flex-row gap-[8px] pt-[2px]">
        <PrimaryButton title={strings.note.conflictReload} onPress={onReload} />
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={strings.note.conflictOverwrite}
          onPress={onOverwrite}
          className="rounded-md border border-neutral-700 px-[14px] py-[10px]"
        >
          <Text className="font-med text-[13px] text-neutral-300">
            {strings.note.conflictOverwrite}
          </Text>
        </Pressable>
      </View>
    </View>
  );
}

/**
 * The design's margin comments, anchored beside the block column rather than to a block.
 *
 * The block bar's comment button opens the composer here: 1c draws the button as "comment on
 * THIS block", and the proto's Comment has no block id, so what it can honestly do is start a
 * comment on the note. The column appears for the composer even when there is nothing in it
 * yet — otherwise the button would look like it did nothing.
 *
 * It is drawn only when the window has the width for it; below that the same thread stacks
 * under the note (StackedComments), which is also the mobile shape.
 */
function CommentsColumn({
  noteId,
  nameOf,
  composing,
  onCloseComposer,
}: {
  noteId: string;
  nameOf: (id: string) => string;
  composing: boolean;
  onCloseComposer: () => void;
}) {
  const comments = useNoteComments(noteId, true);
  const add = useAddNoteComment();
  const [body, setBody] = useState("");
  const rows = comments.data ?? [];

  if (rows.length === 0 && !composing) return null;

  const submit = () => {
    const text = body.trim();
    if (text === "") return;
    add.mutate({ noteId, body: text });
    setBody("");
    onCloseComposer();
  };

  return (
    <View className="flex-none gap-[10px] px-[10px] py-[44px]" style={{ width: PANE.comments }}>
      {composing ? (
        <View className="gap-[8px] rounded-md bg-surface px-[12px] py-[11px]">
          <TextInput
            value={body}
            onChangeText={setBody}
            autoFocus
            multiline
            placeholder={strings.note.commentPlaceholder}
            placeholderTextColor={nocturne.neutral[600]}
            accessibilityLabel={strings.note.addComment}
            onSubmitEditing={submit}
            className="min-h-[54px] font-sans text-[12.5px] leading-[19px] text-fg"
          />
          <View className="flex-row items-center gap-[10px]">
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={strings.common.cancel}
              onPress={() => {
                setBody("");
                onCloseComposer();
              }}
            >
              <Text className="font-sans text-[11.5px] text-neutral-500">{strings.common.cancel}</Text>
            </Pressable>
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={strings.note.addComment}
              accessibilityState={{ disabled: body.trim() === "" }}
              disabled={body.trim() === ""}
              onPress={submit}
              className="ml-auto"
            >
              <Text
                className={`font-med text-[11.5px] ${body.trim() === "" ? "text-neutral-600" : "text-accent"}`}
              >
                {strings.note.addComment}
              </Text>
            </Pressable>
          </View>
        </View>
      ) : null}

      <CommentThread comments={rows} nameOf={nameOf} />
    </View>
  );
}

/**
 * The open comments, then the resolved ones behind a disclosure.
 *
 * Both lists ask ListComments for the resolved rows too. They used to ask for open ones only,
 * which made resolving one-way and half of CommentCard dead code: a resolved comment vanished
 * from the only list that could have un-resolved it, and `comment.resolved` was false wherever
 * the card rendered. Showing them collapsed is the cheaper of the two honest fixes — the
 * contract already carries the flag both ways (ResolveComment takes a bool, not a verb), and
 * the alternative was to delete the branch and leave "resolve" as a one-way door.
 */
function CommentThread({
  comments,
  nameOf,
}: {
  comments: readonly Comment[];
  nameOf: (id: string) => string;
}) {
  const [showResolved, setShowResolved] = useState(false);
  const open = comments.filter((comment) => !comment.resolved);
  const resolved = comments.filter((comment) => comment.resolved);

  return (
    <View className="gap-[10px]">
      {open.map((comment) => (
        <CommentCard key={comment.id} comment={comment} nameOf={nameOf} />
      ))}

      {resolved.length > 0 ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={strings.note.resolved}
          accessibilityState={{ expanded: showResolved }}
          onPress={() => setShowResolved((current) => !current)}
          className="flex-row items-center gap-[6px] py-[2px]"
        >
          <Icon
            name={showResolved ? "caret-up" : "caret-down"}
            size={11}
            color={nocturne.neutral[600]}
          />
          <Text className="font-sans text-[11.5px] text-neutral-500">{strings.note.resolved}</Text>
          <Text className="font-sans text-[11.5px] text-neutral-600">{resolved.length}</Text>
        </Pressable>
      ) : null}

      {showResolved
        ? resolved.map((comment) => (
            <CommentCard key={comment.id} comment={comment} nameOf={nameOf} />
          ))
        : null}
    </View>
  );
}

/**
 * One comment. The control at the foot is a toggle in both directions — ResolveComment takes
 * the value to set, and a resolved card is reachable through the disclosure above, so pressing
 * it there re-opens the thread. The server decides who may: the comment's author or the note's
 * owner, and nobody else. That is a rule this app cannot see (there is no "canResolve" on the
 * message), so the control is drawn for everyone and the refusal is the server's to give.
 */
function CommentCard({ comment, nameOf }: { comment: Comment; nameOf: (id: string) => string }) {
  const resolve = useResolveComment();
  const author = nameOf(comment.authorUserId);
  return (
    <View
      className={`gap-[7px] rounded-md bg-surface px-[12px] py-[11px] ${comment.resolved ? "opacity-60" : ""}`}
    >
      <View className="flex-row items-center gap-[7px]">
        <Avatar name={author} size={20} />
        <Text className="font-med text-[12px] text-fg">{firstName(author)}</Text>
        <Text className="ml-auto font-sans text-[10.5px] text-neutral-600">
          {relative(comment.createdAt)}
        </Text>
      </View>
      <Text className="font-sans text-[12.5px] leading-[19px] text-neutral-300">{comment.body}</Text>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={comment.resolved ? strings.note.resolved : strings.note.resolve}
        accessibilityState={{ checked: comment.resolved }}
        onPress={() => resolve.mutate({ commentId: comment.id, resolved: !comment.resolved })}
      >
        <Text
          className={`font-sans text-[11.5px] ${comment.resolved ? "text-neutral-500" : "text-accent"}`}
        >
          {comment.resolved ? strings.note.resolved : strings.note.resolve}
        </Text>
      </Pressable>
    </View>
  );
}

/**
 * The comment thread stacked under the note: the mobile shape, and the desktop one whenever the
 * window is too narrow for the margin column. `composing` is the block bar's comment button
 * arriving from a layout that has no margin to open — it puts the caret in the field here
 * instead, then hands the flag back so the next press fires again.
 */
function StackedComments({
  noteId,
  nameOf,
  composing = false,
  onComposerShown,
}: {
  noteId: string;
  nameOf: (id: string) => string;
  composing?: boolean;
  onComposerShown?: () => void;
}) {
  const comments = useNoteComments(noteId, true);
  const add = useAddNoteComment();
  const [body, setBody] = useState("");
  const input = useRef<TextInput>(null);
  const rows = comments.data ?? [];

  useEffect(() => {
    if (!composing) return;
    input.current?.focus();
    onComposerShown?.();
  }, [composing, onComposerShown]);

  return (
    <View className="gap-[10px] pt-[24px]">
      <Divider />
      <Text className="font-semi text-[10px] uppercase text-neutral-500" style={{ letterSpacing: 1 }}>
        {strings.note.comments}
      </Text>
      <CommentThread comments={rows} nameOf={nameOf} />
      <View className="flex-row items-center gap-[8px]">
        <TextInput
          ref={input}
          value={body}
          onChangeText={setBody}
          placeholder={strings.note.commentPlaceholder}
          placeholderTextColor={nocturne.neutral[600]}
          accessibilityLabel={strings.note.addComment}
          className="flex-1 rounded-md border border-neutral-800 px-[12px] py-[9px] font-sans text-[13px] text-fg"
        />
        <IconButton
          icon="plus"
          label={strings.note.addComment}
          size={18}
          color={nocturne.accent.DEFAULT}
          disabled={body.trim() === ""}
          onPress={() => {
            add.mutate({ noteId, body: body.trim() });
            setBody("");
          }}
        />
      </View>
    </View>
  );
}

/** The right rail of artboard 1c: outline, activity, who this note is shared with. */
function EditorRail({
  note,
  blocks,
  nameOf,
  onInvite,
}: {
  note: Note;
  blocks: Block[];
  nameOf: (id: string) => string;
  onInvite: () => void;
}) {
  const activity = useActivity(note.id, 12);
  const headings = outline(blocks);

  return (
    <View
      className="flex-none gap-[16px] border-l border-neutral-900 px-[16px] py-[20px]"
      style={{ width: PANE.editorRail }}
    >
      <RailHeading label={strings.note.inThisNote} />
      {headings.length === 0 ? (
        <Text className="font-sans text-[12.5px] text-neutral-600">—</Text>
      ) : (
        <View className="gap-[7px]">
          {headings.map((heading, index) => (
            <Text
              key={heading.id}
              numberOfLines={1}
              className={`font-sans text-[12.5px] ${index === 0 ? "text-accent-300" : "text-neutral-400"}`}
              style={{ paddingLeft: (heading.level - 1) * 10 }}
            >
              {heading.text}
            </Text>
          ))}
        </View>
      )}

      <Divider />
      <RailHeading label={strings.note.activity} />
      <View className="gap-[11px]">
        {(activity.data ?? []).length === 0 ? (
          <Text className="font-sans text-[12px] text-neutral-600">{strings.activity.empty}</Text>
        ) : null}
        {(activity.data ?? []).map((entry) => (
          <View key={entry.id} className="flex-row gap-[8px]">
            <Avatar name={nameOf(entry.actorUserId)} size={18} />
            <Text className="flex-1 font-sans text-[12px] leading-[18px] text-neutral-400">
              {sentenceFor(entry, nameOf)} · {relative(entry.createdAt)}
            </Text>
          </View>
        ))}
      </View>

      <Divider />
      <RailHeading label={strings.note.sharedWith} />
      <View className="gap-[9px]">
        <View className="flex-row items-center gap-[8px]">
          <Avatar name={nameOf(note.ownerUserId)} size={20} />
          <Text className="shrink font-sans text-[12.5px] text-fg" numberOfLines={1}>
            {nameOf(note.ownerUserId)}
          </Text>
          <Text className="ml-auto font-sans text-[11px] text-neutral-600">{strings.note.owner}</Text>
        </View>
        {note.shares.map((share) => (
          <View key={share.id} className="flex-row items-center gap-[8px]">
            {share.subject === ShareSubject.FAMILY ? (
              <View className="h-[20px] w-[20px] items-center justify-center rounded-full bg-accent-800">
                <Icon name="users-three" size={11} color={nocturne.accent[200]} />
              </View>
            ) : (
              <Avatar name={nameOf(share.memberUserId)} size={20} />
            )}
            <Text className="shrink font-sans text-[12.5px] text-fg" numberOfLines={1}>
              {share.subject === ShareSubject.FAMILY
                ? strings.share.wholeFamily
                : nameOf(share.memberUserId)}
            </Text>
            <Text className="ml-auto font-sans text-[11px] text-neutral-600">
              {share.permission === SharePermission.EDIT ? strings.note.canEdit : strings.note.canView}
            </Text>
          </View>
        ))}
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={strings.note.share}
          onPress={onInvite}
          className="flex-row items-center gap-[8px] pt-[2px]"
        >
          <View className="h-[20px] w-[20px] items-center justify-center rounded-full border border-dashed border-neutral-700">
            <Icon name="plus" size={10} color={nocturne.neutral[500]} />
          </View>
          <Text className="font-sans text-[12.5px] text-neutral-500">{strings.note.share}</Text>
        </Pressable>
      </View>
    </View>
  );
}

function RailHeading({ label }: { label: string }) {
  return (
    <Text className="font-semi text-[10px] uppercase text-neutral-500" style={{ letterSpacing: 1 }}>
      {label}
    </Text>
  );
}

/** The server sends kinds, not prose — the proto says so — so the sentence is written here. */
function sentenceFor(entry: Activity, nameOf: (id: string) => string): string {
  const who = firstName(nameOf(entry.actorUserId)) || "Someone";
  switch (entry.kind) {
    case ActivityKind.CREATED:
      return strings.activity.created(who);
    case ActivityKind.EDITED:
      return strings.activity.edited(who);
    case ActivityKind.TASK_CHECKED:
      return strings.activity.taskChecked(who, entry.detail);
    case ActivityKind.TASK_UNCHECKED:
      return strings.activity.taskUnchecked(who, entry.detail);
    case ActivityKind.COMMENTED:
      return strings.activity.commented(who);
    case ActivityKind.SHARED:
      return strings.activity.shared(who, entry.detail);
    default:
      return strings.activity.unknown(who);
  }
}

/**
 * The avatar stack. In the mock it was "who is editing right now"; here it is who the note is
 * shared with, which is the true version of the same picture.
 */
function shareNamesOf(note: Note, nameOf: (id: string) => string): string[] {
  const names = [nameOf(note.ownerUserId)];
  for (const share of note.shares) {
    if (share.subject === ShareSubject.FAMILY) continue;
    const name = nameOf(share.memberUserId);
    if (name !== "" && !names.includes(name)) names.push(name);
  }
  return names.filter((name) => name !== "");
}

function breadcrumb(notebookName: string | undefined): string {
  return notebookName ? `${notebookName} /` : "";
}
