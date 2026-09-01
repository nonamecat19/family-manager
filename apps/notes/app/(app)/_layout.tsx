import { Code } from "@connectrpc/connect";
import {
  toDisplayError,
  useCreateNote,
  useCreateNotebook,
  useDeleteNotebook,
  useFamily,
  useNotebooks,
  useNotes,
  useUpdateNotebook,
} from "@fm/api";
import type { NoteListFilters } from "@fm/api";
import { NoteSort, type Note, type Notebook } from "@fm/sdk/notes/v1/notes_pb";
import { Slot, Tabs, usePathname, useRouter, useSegments } from "expo-router";
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import {
  ActivityIndicator,
  Modal,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
  useWindowDimensions,
  type ColorValue,
} from "react-native";

import { bucket, relative, stamp, strings } from "../../components/i18n/index.ts";
import {
  Avatar,
  AvatarStack,
  Chip,
  Divider,
  EmptyState,
  Icon,
  IconButton,
  Kbd,
  Pane,
  PrimaryButton,
  Rail,
  Screen,
  nocturne,
  type IconName,
} from "../../components/nocturne/index.ts";
import { Palette, useDesktopShortcuts } from "../../components/palette.tsx";
import { useMemberDirectory } from "../../components/share.tsx";
import { makeBlock } from "../../components/editor/index.ts";
import type { QueuedNote } from "../../components/offline/index.ts";

/**
 * The app shell. ONE route tree, two layouts.
 *
 * At >= 1024px (desktop, and the Tauri window) the design's three panes are a LAYOUT, not a
 * screen: the rail and the note list live here and stay put while `<Slot/>` swaps the third
 * pane. Below that width the same routes hang off a tab bar and each one is full-bleed.
 *
 * Everything the two layouts share — which list the rail has selected, the note rows, the
 * palette — is defined in this file and imported by the screens. It is the only file both
 * layouts render, so it is the only place that can hold their common state without a
 * component tree that exists twice.
 */

export const DESKTOP_MIN_WIDTH = 1024;

export function useIsDesktop(): boolean {
  const { width } = useWindowDimensions();
  return width >= DESKTOP_MIN_WIDTH;
}

/**
 * The fixed widths of the desktop chrome, and the narrowest the editor's text column may get.
 *
 * These are numbers rather than four `w-[…]` classes scattered across two files because the
 * panes have to be decided by arithmetic against each other: 236 + 322 + 268 + 200 is 1026px
 * of chrome on its own, one pixel past the width at which the desktop layout switches on. In
 * React Native `flexShrink` defaults to 0, so a chrome sum wider than the window does not
 * squeeze anything — the flexible column between the panes is what collapses, to nothing.
 */
export const PANE = {
  rail: 236,
  list: 322,
  editorRail: 268,
  comments: 200,
  /** The design's measure is 560. Below this the column has stopped being a page. */
  minColumn: 460,
} as const;

export interface PaneFit {
  desktop: boolean;
  /** The shell's note-list pane — the middle column of artboard 1a. */
  list: boolean;
  /** Artboard 1c's outline / activity / shared-with rail. */
  editorRail: boolean;
  /** The comment margin. When it does not fit, the thread stacks under the note instead. */
  comments: boolean;
}

/**
 * Which desktop panes this window can afford.
 *
 * On a list route there are only two panes (236 + 322) and they always fit. On the editor route
 * the width is spent in the order the design spends it — text column first, then 1c's right
 * rail, then the note list, then the comment margin. With these numbers that is: the rail and
 * the column below 1286px, the note list from 1286, the comment margin from 1486.
 *
 * Dropping the note list first is not a compromise. Artboard 1c is a FULL-WINDOW editor in the
 * mock — 1240px of header, text column and a 268px rail, with no note list beside it — so up to
 * 1286 this app draws what the mock draws, and the back arrow in the editor header is the way
 * back to the list. Nothing is unreachable at any width: the comment thread stacks under the
 * note when its margin is gone, and the outline the right rail draws is the note itself.
 */
export function usePaneFit(): PaneFit {
  const { width } = useWindowDimensions();
  const pathname = usePathname();

  if (width < DESKTOP_MIN_WIDTH) {
    return { desktop: false, list: false, editorRail: false, comments: false };
  }
  if (!pathname.startsWith("/note/")) {
    return { desktop: true, list: true, editorRail: false, comments: false };
  }

  // The rail is the app's navigation and the column is the note; everything after those two
  // is bought with what is left, in priority order.
  let spare = width - PANE.rail - PANE.minColumn;
  const editorRail = spare >= PANE.editorRail;
  if (editorRail) spare -= PANE.editorRail;
  const list = spare >= PANE.list;
  if (list) spare -= PANE.list;
  // The margin waits for everything ahead of it to have a place, so the panes only ever
  // appear as the window grows. A column that turns up at 1240 and disappears at 1290 —
  // because the wider window spent its width on the list instead — is a flicker, not a layout.
  return { desktop: true, list, editorRail, comments: list && spare >= PANE.comments };
}

/** Which list the rail has selected. `notebook` reads `notebookId` alongside it. */
export type ListView = "all" | "shared" | "recent" | "starred" | "archive" | "notebook";

export interface ShellValue {
  view: ListView;
  notebookId: string;
  select: (view: ListView, notebookId?: string) => void;
  openPalette: () => void;
  openCapture: () => void;
  newNote: (title?: string) => void;
  density: "cards" | "dense";
  setDensity: (density: "cards" | "dense") => void;
  /** The order ListNotes is asked for. One of the contract's NoteSort values, nothing else. */
  sort: NoteSort;
  setSort: (sort: NoteSort) => void;
  captureVisible: boolean;
  closeCapture: () => void;
  notebooks: readonly Notebook[];
}

const ShellContext = createContext<ShellValue | null>(null);

export function useShell(): ShellValue {
  const value = useContext(ShellContext);
  if (!value) throw new Error("useShell used outside the (app) shell");
  return value;
}

/** The filters a rail selection turns into. One place, so the two layouts cannot disagree. */
export function filtersFor(
  view: ListView,
  notebookId: string,
  sort: NoteSort = NoteSort.UPDATED,
): NoteListFilters {
  switch (view) {
    case "shared":
      return { sharedOnly: true, sort };
    case "starred":
      return { starredOnly: true, sort };
    // "Recent" is an order, not a filter: it ignores the list's own sort by definition.
    case "recent":
      return { sort: NoteSort.UPDATED, pageSize: 20 };
    case "archive":
      return { archivedOnly: true, sort };
    case "notebook":
      return { notebookId, sort };
    default:
      return { sort };
  }
}

/** The sort control's menu, in the order the design lists it. */
export const SORT_OPTIONS: readonly { sort: NoteSort; label: string }[] = [
  { sort: NoteSort.UPDATED, label: strings.list.sortUpdated },
  { sort: NoteSort.CREATED, label: strings.list.sortCreated },
  { sort: NoteSort.TITLE, label: strings.list.sortTitle },
];

export default function AppLayout() {
  return <Gate />;
}

/**
 * The family gate, the same shape apps/recipes uses: notes hang off a family, and every list
 * in this app is scoped by the token's family_id claim. FailedPrecondition means "you have an
 * account but no family yet", which is a screen, not an error.
 */
function Gate() {
  const family = useFamily();
  const router = useRouter();
  const segments = useSegments();
  const isOnboarding = segments[segments.length - 1] === "onboarding";

  if (family.isPending) return <Booting label={strings.gate.settingUp} />;

  if (family.isError) {
    const code = (family.error as { code?: Code }).code;
    if (code === Code.FailedPrecondition) {
      // Onboarding lives under this same gate, so without this branch pushing to it only
      // changes the URL — this component still short-circuits before any outlet renders.
      if (isOnboarding) return <Slot />;
      return (
        <Screen>
          <View className="flex-1 justify-center gap-[18px] px-[22px]">
            <Text className="font-semi text-[28px] text-fg">{strings.gate.noFamilyTitle}</Text>
            <Text className="font-sans text-[15px] leading-[23px] text-neutral-400">
              {strings.gate.noFamilyBody}
            </Text>
            <PrimaryButton
              title={strings.gate.createFamily}
              onPress={() => router.push("/(app)/onboarding")}
            />
          </View>
        </Screen>
      );
    }
    const shown = toDisplayError(family.error, strings.common.loadFailed);
    return (
      <Screen>
        <View className="flex-1 justify-center gap-[18px] px-[22px]">
          <Text className="font-semi text-[28px] text-fg">{strings.gate.errorTitle}</Text>
          <Text className="font-sans text-[15px] leading-[23px] text-neutral-400">{shown.message}</Text>
          {shown.reference ? (
            <Text className="font-sans text-[13px] text-neutral-600">
              {strings.common.errorReference(shown.reference)}
            </Text>
          ) : null}
          <PrimaryButton title={strings.common.tryAgain} onPress={() => void family.refetch()} />
        </View>
      </Screen>
    );
  }

  return <Shell />;
}

function Shell() {
  const desktop = useIsDesktop();
  const router = useRouter();
  const createNote = useCreateNote();

  const [view, setView] = useState<ListView>("all");
  const [notebookId, setNotebookId] = useState("");
  const [paletteVisible, setPaletteVisible] = useState(false);
  const [captureVisible, setCaptureVisible] = useState(false);
  const [density, setDensity] = useState<"cards" | "dense">("cards");
  const [sort, setSort] = useState<NoteSort>(NoteSort.UPDATED);
  // Whether ListNotebooks is asked for the archived ones too. It is not `view === "archive"`:
  // picking an archived notebook out of the archive moves the view to "notebook", and a query
  // keyed on the view would unload the very notebook that was just selected — leaving the rail
  // row gone and the list pane titled "All notes". It stays on until the user leaves for a
  // view that is not about the archive.
  const [showArchivedNotebooks, setShowArchivedNotebooks] = useState(false);

  // ONE notebook query for the whole shell. The rail draws it, the list pane titles itself
  // from it, and the move/capture sheets offer it — three panes reading two different lists is
  // how they end up disagreeing about which notebooks exist.
  const notebooks = useNotebooks(showArchivedNotebooks);

  const select = useCallback((next: ListView, id = "") => {
    setView(next);
    setNotebookId(id);
    setShowArchivedNotebooks((current) => next === "archive" || (next === "notebook" && current));
  }, []);

  const newNote = useCallback(
    (title = "") => {
      createNote.mutate(
        { notebookId, title, blocks: [makeBlock()] },
        { onSuccess: (note) => note && router.push(`/(app)/note/${note.id}`) },
      );
    },
    [createNote, notebookId, router],
  );

  const openPalette = useCallback(() => setPaletteVisible(true), []);
  const openCapture = useCallback(() => setCaptureVisible(true), []);
  const closeCapture = useCallback(() => setCaptureVisible(false), []);

  useDesktopShortcuts({ onPalette: openPalette, onNewNote: () => newNote() });

  const value = useMemo<ShellValue>(
    () => ({
      view,
      notebookId,
      select,
      openPalette,
      openCapture,
      closeCapture,
      captureVisible,
      newNote,
      density,
      setDensity,
      sort,
      setSort,
      notebooks: notebooks.data ?? [],
    }),
    [
      view,
      notebookId,
      select,
      openPalette,
      openCapture,
      closeCapture,
      captureVisible,
      newNote,
      density,
      sort,
      notebooks.data,
    ],
  );

  return (
    <ShellContext.Provider value={value}>
      {desktop ? <DesktopShell /> : <MobileTabs />}
      <Palette
        visible={paletteVisible}
        onClose={() => setPaletteVisible(false)}
        onOpenNote={(id) => router.push(`/(app)/note/${id}`)}
        onOpenNotebook={(id) => {
          select("notebook", id);
          router.push(`/(app)/notebook/${id}`);
        }}
        onCreateNote={(title) => newNote(title)}
      />
    </ShellContext.Provider>
  );
}

/* ------------------------------------------------------------------ desktop */

function DesktopShell() {
  const panes = usePaneFit();
  return (
    <View className="flex-1 flex-row bg-bg">
      <RailSidebar />
      {panes.list ? <NoteListPane /> : null}
      <View className="min-w-0 flex-1">
        <Slot />
      </View>
    </View>
  );
}

export function RailSidebar() {
  const shell = useShell();
  const router = useRouter();
  const all = useNotes({});
  const shared = useNotes({ sharedOnly: true });
  const { members } = useMemberDirectory();

  const createNotebook = useCreateNotebook();
  const updateNotebook = useUpdateNotebook();
  const deleteNotebook = useDeleteNotebook();

  const [creating, setCreating] = useState(false);
  const [renaming, setRenaming] = useState<Notebook | null>(null);
  const [menuFor, setMenuFor] = useState<Notebook | null>(null);
  const [deleting, setDeleting] = useState<Notebook | null>(null);

  // The shell's list, which carries the archived notebooks whenever the archive is in play —
  // they are the only way back out of the archive, so the rail has to be able to draw them.
  const tree = useMemo(() => buildTree(shell.notebooks), [shell.notebooks]);

  const failed = createNotebook.isError || updateNotebook.isError || deleteNotebook.isError;

  return (
    <Rail
      className="flex-none gap-[14px] px-[12px] py-[16px]"
      style={{ width: PANE.rail }}
    >
      <View className="flex-row items-center gap-[9px] px-[4px]">
        <View className="h-[22px] w-[22px] items-center justify-center rounded-md border border-accent">
          <Text className="font-semi text-[11px] text-accent">C</Text>
        </View>
        <Text className="font-med text-[15px] text-fg">{strings.app.name}</Text>
      </View>

      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.rail.search}
        onPress={shell.openPalette}
        className="h-[32px] flex-row items-center gap-[8px] rounded-md border border-neutral-800 bg-surface px-[10px]"
      >
        <Icon name="magnifying-glass" size={14} color={nocturne.neutral[500]} />
        <Text className="font-sans text-[13px] text-neutral-500">{strings.rail.search}</Text>
        <View className="ml-auto">
          <Kbd>⌘K</Kbd>
        </View>
      </Pressable>

      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.rail.newNote}
        onPress={() => shell.newNote()}
        className="h-[34px] flex-row items-center justify-center gap-[7px] rounded-md border border-accent"
      >
        <Icon name="plus" size={14} color={nocturne.accent.DEFAULT} />
        <Text className="font-med text-[13px] text-accent">{strings.rail.newNote}</Text>
        <Kbd>⌘N</Kbd>
      </Pressable>

      <View className="gap-[1px]">
        <RailRow
          icon="notebook"
          label={strings.rail.allNotes}
          count={all.data?.length}
          active={shell.view === "all"}
          onPress={() => {
            shell.select("all");
            router.push("/(app)");
          }}
        />
        <RailRow
          icon="users-three"
          label={strings.rail.sharedWithMe}
          count={shared.data?.length}
          active={shell.view === "shared"}
          onPress={() => {
            shell.select("shared");
            router.push("/(app)/shared");
          }}
        />
        <RailRow
          icon="clock-counter-clockwise"
          label={strings.rail.recent}
          active={shell.view === "recent"}
          onPress={() => {
            shell.select("recent");
            router.push("/(app)");
          }}
        />
        <RailRow
          icon="star"
          label={strings.rail.starred}
          active={shell.view === "starred"}
          onPress={() => {
            shell.select("starred");
            router.push("/(app)");
          }}
        />
      </View>

      <Divider className="my-[2px]" />

      <View className="flex-row items-center px-[10px]">
        <Text className="font-semi text-[10px] uppercase text-neutral-500" style={{ letterSpacing: 1 }}>
          {strings.rail.notebooks}
        </Text>
        <View className="ml-auto">
          <MiniButton icon="plus" label={strings.rail.newNotebook} size={13} onPress={() => setCreating(true)} />
        </View>
      </View>

      <ScrollView className="grow-0" showsVerticalScrollIndicator={false}>
        <View className="gap-[1px]">
          {tree.map((node) => (
            <View key={node.notebook.id}>
              <RailRow
                icon="folder-simple"
                iconColor={nocturne.accent[400]}
                label={node.notebook.name}
                count={node.notebook.noteCount}
                active={shell.view === "notebook" && shell.notebookId === node.notebook.id}
                onPress={() => {
                  shell.select("notebook", node.notebook.id);
                  router.push(`/(app)/notebook/${node.notebook.id}`);
                }}
                onMenu={() => setMenuFor(node.notebook)}
              />
              {node.children.map((child) => (
                <RailRow
                  key={child.id}
                  label={child.name}
                  nested
                  count={child.noteCount}
                  active={shell.view === "notebook" && shell.notebookId === child.id}
                  onPress={() => {
                    shell.select("notebook", child.id);
                    router.push(`/(app)/notebook/${child.id}`);
                  }}
                  onMenu={() => setMenuFor(child)}
                />
              ))}
            </View>
          ))}
        </View>
      </ScrollView>

      <RailRow
        icon="archive"
        label={strings.rail.archive}
        active={shell.view === "archive"}
        onPress={() => {
          shell.select("archive");
          router.push("/(app)");
        }}
      />

      {failed ? (
        <Text className="px-[10px] font-sans text-[11px] text-neutral-500">
          {strings.rail.notebookFailed}
        </Text>
      ) : null}

      <NotebookDialog
        // Both dialogs are remounted per opening so the field starts empty, or from the
        // notebook being renamed, rather than from whatever was typed the last time.
        key={creating ? "create-open" : "create-closed"}
        visible={creating}
        title={strings.rail.createNotebook}
        confirmLabel={strings.rail.create}
        initialName=""
        busy={createNotebook.isPending}
        onClose={() => setCreating(false)}
        onSubmit={(name) => {
          createNotebook.mutate({ name });
          setCreating(false);
        }}
      />

      <NotebookDialog
        key={renaming?.id ?? "rename"}
        visible={renaming !== null}
        title={strings.rail.renameNotebook}
        confirmLabel={strings.rail.rename}
        initialName={renaming?.name ?? ""}
        busy={updateNotebook.isPending}
        onClose={() => setRenaming(null)}
        onSubmit={(name) => {
          if (renaming) {
            updateNotebook.mutate({
              notebookId: renaming.id,
              name,
              parentId: renaming.parentId,
              archived: renaming.archived,
            });
          }
          setRenaming(null);
        }}
      />

      <ActionSheet
        visible={menuFor !== null}
        title={menuFor?.name ?? strings.rail.notebookActions}
        onClose={() => setMenuFor(null)}
        actions={
          menuFor
            ? [
                {
                  key: "rename",
                  label: strings.rail.rename,
                  icon: "pencil-simple",
                  onPress: () => setRenaming(menuFor),
                },
                {
                  key: "archive",
                  label: menuFor.archived ? strings.rail.restoreNotebook : strings.rail.archiveNotebook,
                  icon: "archive",
                  onPress: () =>
                    updateNotebook.mutate({
                      notebookId: menuFor.id,
                      name: menuFor.name,
                      parentId: menuFor.parentId,
                      archived: !menuFor.archived,
                    }),
                },
                {
                  key: "delete",
                  label: strings.rail.deleteNotebook,
                  icon: "trash",
                  tone: "danger",
                  onPress: () => setDeleting(menuFor),
                },
              ]
            : []
        }
      />

      <ActionSheet
        visible={deleting !== null}
        title={strings.rail.deleteNotebookConfirm}
        onClose={() => setDeleting(null)}
        actions={[
          {
            key: "confirm",
            label: strings.rail.deleteNotebook,
            icon: "trash",
            tone: "danger",
            onPress: () => {
              if (!deleting) return;
              deleteNotebook.mutate(deleting.id);
              if (shell.notebookId === deleting.id) shell.select("all");
            },
          },
        ]}
      />

      <View className="mt-auto flex-row items-center gap-[8px] px-[10px] pt-[8px]">
        <AvatarStack names={members.map((m) => m.displayName || m.email)} size={22} max={3} />
        <Text className="font-sans text-[11px] text-neutral-500">
          {strings.rail.familyCount(members.length)}
        </Text>
        <View className="ml-auto">
          <IconButton
            icon="gear-six"
            label={strings.rail.settings}
            size={15}
            color={nocturne.neutral[500]}
            onPress={() => router.push("/(app)/settings")}
          />
        </View>
      </View>
    </Rail>
  );
}

function RailRow({
  icon,
  iconColor,
  label,
  count,
  active = false,
  nested = false,
  onPress,
  onMenu,
}: {
  icon?: IconName;
  iconColor?: string;
  label: string;
  count?: number;
  active?: boolean;
  nested?: boolean;
  onPress: () => void;
  /** Notebook rows carry rename/archive/delete. The design draws it on hover; there is no
   *  hover on a touch screen, so it is a real control on the row and a long press as well. */
  onMenu?: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected: active }}
      onPress={onPress}
      onLongPress={onMenu}
      className={`h-[30px] flex-row items-center gap-[9px] rounded-md ${nested ? "pl-[22px] pr-[10px]" : "px-[10px]"}`}
      style={active ? { backgroundColor: nocturne.accent[900] } : undefined}
    >
      {icon ? (
        <Icon
          name={icon}
          size={15}
          color={active ? nocturne.accent[300] : (iconColor ?? nocturne.neutral[500])}
          weight={active ? "fill" : "regular"}
        />
      ) : nested ? (
        <View className="h-[5px] w-[5px] rounded-full bg-neutral-600" />
      ) : null}
      <Text
        numberOfLines={1}
        className={`shrink font-sans text-[13px] ${active ? "text-accent-300" : "text-neutral-300"}`}
      >
        {label}
      </Text>
      {typeof count === "number" && count > 0 ? (
        <Text className="ml-auto font-sans text-[11px] text-neutral-600">{count}</Text>
      ) : null}
      {onMenu ? (
        <View className={typeof count === "number" && count > 0 ? "" : "ml-auto"}>
          <MiniButton icon="dots-three" label={strings.rail.notebookActions} size={14} onPress={onMenu} />
        </View>
      ) : null}
    </Pressable>
  );
}

/**
 * A control small enough to sit inside a 30px rail row. The kit's IconButton is a 36px tap
 * target and would stretch the row it belongs to; this one leans on hitSlop instead.
 */
function MiniButton({
  icon,
  label,
  size,
  onPress,
}: {
  icon: IconName;
  label: string;
  size: number;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      hitSlop={10}
      className="h-[20px] w-[20px] items-center justify-center rounded-sm"
    >
      <Icon name={icon} size={size} color={nocturne.neutral[600]} />
    </Pressable>
  );
}

/* ------------------------------------------------------------------- sheets */

const SCRIM = { backgroundColor: nocturne.bg, opacity: 0.62 } as const;

export interface SheetAction {
  key: string;
  label: string;
  icon?: IconName;
  /** Draws the check the design puts beside the chosen option in a menu. */
  selected?: boolean;
  tone?: "default" | "danger";
  onPress: () => void;
}

/**
 * A menu, as a sheet. The design draws its menus as popovers anchored to the control that
 * opened them; anchoring means measuring a view and doing arithmetic against the window, and
 * this app runs the same tree on a phone where a popover is the wrong shape anyway. One
 * centred card, the same on both.
 *
 * It lives in the shell rather than the Nocturne kit because it is a composition of kit parts
 * for these two screens, not a primitive: both the rail's notebook actions and the editor's
 * overflow menu are this list of rows and nothing more.
 */
export function ActionSheet({
  visible,
  title,
  actions,
  onClose,
  children,
}: {
  visible: boolean;
  title: string;
  actions: readonly SheetAction[];
  onClose: () => void;
  children?: ReactNode;
}) {
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.common.close}
        onPress={onClose}
        className="absolute inset-0"
        style={SCRIM}
      />
      <View className="flex-1 items-center justify-center px-[24px]" pointerEvents="box-none">
        <View className="w-full max-w-[320px] gap-[6px] rounded-lg bg-surface px-[10px] py-[12px]">
          <Text className="px-[8px] pb-[2px] font-sans text-[12px] leading-[18px] text-neutral-500">
            {title}
          </Text>
          {children}
          {actions.map((action) => (
            <Pressable
              key={action.key}
              accessibilityRole="button"
              accessibilityLabel={action.label}
              accessibilityState={{ selected: action.selected ?? false }}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              className="h-[36px] flex-row items-center gap-[10px] rounded-md px-[8px]"
            >
              {action.icon ? (
                <Icon
                  name={action.icon}
                  size={15}
                  color={action.tone === "danger" ? nocturne.neutral[400] : nocturne.accent[400]}
                />
              ) : null}
              <Text
                className={`font-sans text-[13.5px] ${
                  action.tone === "danger" ? "text-neutral-200" : "text-fg"
                }`}
              >
                {action.label}
              </Text>
              {action.selected ? (
                <View className="ml-auto">
                  <Icon name="check" size={13} color={nocturne.accent.DEFAULT} />
                </View>
              ) : null}
            </Pressable>
          ))}
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={strings.common.cancel}
            onPress={onClose}
            className="h-[34px] items-center justify-center rounded-md"
          >
            <Text className="font-sans text-[13px] text-neutral-500">{strings.common.cancel}</Text>
          </Pressable>
        </View>
      </View>
    </Modal>
  );
}

/** Create or rename a notebook: one field, because a notebook is a name and a parent. */
function NotebookDialog({
  visible,
  title,
  confirmLabel,
  initialName,
  busy,
  onClose,
  onSubmit,
}: {
  visible: boolean;
  title: string;
  confirmLabel: string;
  initialName: string;
  busy: boolean;
  onClose: () => void;
  onSubmit: (name: string) => void;
}) {
  const [name, setName] = useState(initialName);

  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.common.close}
        onPress={onClose}
        className="absolute inset-0"
        style={SCRIM}
      />
      <View className="flex-1 items-center justify-center px-[24px]" pointerEvents="box-none">
        <View className="w-full max-w-[320px] gap-[12px] rounded-lg bg-surface p-[16px]">
          <Text className="font-med text-[15px] text-fg">{title}</Text>
          <TextInput
            value={name}
            onChangeText={setName}
            autoFocus
            placeholder={strings.rail.notebookNamePlaceholder}
            placeholderTextColor={nocturne.neutral[600]}
            accessibilityLabel={strings.rail.notebookName}
            onSubmitEditing={() => {
              if (name.trim() !== "") onSubmit(name.trim());
            }}
            className="rounded-md border border-neutral-800 px-[12px] py-[9px] font-sans text-[14px] text-fg"
          />
          <View className="flex-row items-center gap-[8px]">
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={strings.common.cancel}
              onPress={onClose}
              className="rounded-md border border-neutral-800 px-[14px] py-[9px]"
            >
              <Text className="font-med text-[13px] text-neutral-300">{strings.common.cancel}</Text>
            </Pressable>
            <View className="ml-auto">
              <PrimaryButton
                title={confirmLabel}
                disabled={busy || name.trim() === ""}
                onPress={() => onSubmit(name.trim())}
              />
            </View>
          </View>
        </View>
      </View>
    </Modal>
  );
}

interface TreeNode {
  notebook: Notebook;
  children: Notebook[];
}

/** The design's one-level tree. The column allows deeper; the rail draws two. */
function buildTree(notebooks: readonly Notebook[]): TreeNode[] {
  const roots = notebooks.filter((n) => n.parentId === "");
  return roots.map((notebook) => ({
    notebook,
    children: notebooks.filter((n) => n.parentId === notebook.id),
  }));
}

/** The middle pane: only ever drawn on desktop, where the list is chrome rather than a screen. */
function NoteListPane() {
  const shell = useShell();
  const router = useRouter();
  const pathname = usePathname();
  const notes = useNotes(filtersFor(shell.view, shell.notebookId, shell.sort));
  const selectedId = pathname.startsWith("/note/") ? pathname.slice("/note/".length) : "";
  const rows = notes.data ?? [];

  const notebook = shell.notebooks.find((n) => n.id === shell.notebookId);
  const title = shell.view === "notebook" ? (notebook?.name ?? strings.list.title) : headingFor(shell.view);

  return (
    <Pane
      tone="list"
      className="flex-none border-x border-neutral-900"
      style={{ width: PANE.list }}
    >
      <View className="flex-row items-center gap-[8px] px-[16px] pb-[12px] pt-[16px]">
        <Text className="font-med text-[17px] text-fg">{title}</Text>
        <Text className="font-sans text-[11px] text-neutral-600">
          {strings.list.noteCount(rows.length)}
        </Text>
        <View className="ml-auto flex-row items-center gap-[2px]">
          <SortButton />
          {/* 1b is 1a at a different density, so the toggle is a control of its own rather
              than a second meaning bolted onto the sort glyph. */}
          <IconButton
            icon={shell.density === "dense" ? "list-bullets" : "rows"}
            label={strings.list.density}
            size={15}
            color={nocturne.neutral[500]}
            onPress={() => shell.setDensity(shell.density === "dense" ? "cards" : "dense")}
          />
          <IconButton
            icon="funnel-simple"
            label={strings.list.filter}
            size={15}
            color={nocturne.neutral[500]}
            onPress={shell.openPalette}
          />
        </View>
      </View>

      <NoteListBody
        notes={rows}
        loading={notes.isPending}
        density={shell.density}
        selectedId={selectedId}
        onOpen={(id) => router.push(`/(app)/note/${id}`)}
        emptyTitle={emptyTitleFor(shell.view)}
        emptyBody={emptyBodyFor(shell.view)}
        onNewNote={() => shell.newNote()}
      />
    </Pane>
  );
}

/**
 * The sort control. The menu is exactly the NoteSort values the contract offers — the app has
 * no client-side ordering of its own, because a sort the server does not know about breaks the
 * moment the list is paged.
 */
function SortButton() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <IconButton
        icon="sort-ascending"
        label={strings.list.sort}
        size={15}
        color={nocturne.neutral[500]}
        onPress={() => setOpen(true)}
      />
      <SortSheet visible={open} onClose={() => setOpen(false)} />
    </>
  );
}

export function SortSheet({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const shell = useShell();
  return (
    <ActionSheet
      visible={visible}
      title={strings.list.sortBy}
      onClose={onClose}
      actions={SORT_OPTIONS.map((option) => ({
        key: String(option.sort),
        label: option.label,
        selected: shell.sort === option.sort,
        onPress: () => shell.setSort(option.sort),
      }))}
    />
  );
}

function headingFor(view: ListView): string {
  if (view === "shared") return strings.rail.sharedWithMe;
  if (view === "starred") return strings.rail.starred;
  if (view === "recent") return strings.rail.recent;
  if (view === "archive") return strings.rail.archive;
  return strings.list.title;
}

function emptyTitleFor(view: ListView): string {
  if (view === "shared") return strings.list.emptySharedTitle;
  if (view === "archive") return strings.list.emptyArchiveTitle;
  return strings.list.emptyTitle;
}

function emptyBodyFor(view: ListView): string {
  if (view === "shared") return strings.list.emptySharedBody;
  if (view === "archive") return strings.list.emptyArchiveBody;
  return strings.list.emptyBody;
}

/* --------------------------------------------------------------- note rows */

export interface NoteListBodyProps {
  notes: readonly Note[];
  loading: boolean;
  density: "cards" | "dense";
  selectedId?: string;
  onOpen: (noteId: string) => void;
  emptyTitle: string;
  emptyBody: string;
  /** Omitted where an empty list is not an invitation to write — the archive. */
  onNewNote?: () => void;
  /** Captures that have not reached the server yet, drawn above the real rows. */
  queued?: readonly QueuedNote[];
}

/**
 * The note list, shared by the desktop pane and the mobile screen. Two densities: the
 * comfortable card of artboard 1a and the one-line row of 1b — 1b is this same list at a
 * different density, not a second layout, so it is a prop rather than a component.
 */
export function NoteListBody({
  notes,
  loading,
  density,
  selectedId = "",
  onOpen,
  emptyTitle,
  emptyBody,
  onNewNote,
  queued = [],
}: NoteListBodyProps) {
  const { nameOf } = useMemberDirectory();

  if (loading) {
    return (
      <View className="items-center py-[40px]">
        <ActivityIndicator color={nocturne.accent.DEFAULT} />
      </View>
    );
  }

  if (notes.length === 0 && queued.length === 0) {
    return (
      <EmptyState
        title={emptyTitle}
        body={emptyBody}
        icon="notebook"
        action={onNewNote ? { label: strings.list.newNote, onPress: onNewNote } : undefined}
      />
    );
  }

  const groups = groupByRecency(notes);

  return (
    <ScrollView showsVerticalScrollIndicator={false}>
      {queued.map((item) => (
        <View key={item.clientId} className="border-b border-neutral-900 px-[16px] py-[13px]">
          <View className="flex-row items-center gap-[7px]">
            <Text className="font-med text-[14px] text-fg" numberOfLines={1}>
              {item.title || strings.common.untitled}
            </Text>
            <View className="ml-auto h-[7px] w-[7px] rounded-full bg-neutral-600" />
          </View>
          <Text className="pt-[4px] font-sans text-[11px] text-neutral-600">
            {strings.list.notSynced}
          </Text>
        </View>
      ))}

      {groups.map((group) => (
        <View key={group.label}>
          {density === "dense" ? (
            <Text className="px-[16px] pb-[4px] pt-[14px] font-semi text-[10px] uppercase text-neutral-600" style={{ letterSpacing: 1 }}>
              {group.label}
            </Text>
          ) : null}
          {group.notes.map((note) =>
            density === "dense" ? (
              <DenseRow
                key={note.id}
                note={note}
                selected={note.id === selectedId}
                onPress={() => onOpen(note.id)}
              />
            ) : (
              <CardRow
                key={note.id}
                note={note}
                editorName={nameOf(note.lastEditedByUserId)}
                selected={note.id === selectedId}
                onPress={() => onOpen(note.id)}
              />
            ),
          )}
        </View>
      ))}
    </ScrollView>
  );
}

function CardRow({
  note,
  editorName,
  selected,
  onPress,
}: {
  note: Note;
  editorName: string;
  selected: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={note.title || strings.common.untitled}
      accessibilityState={{ selected }}
      onPress={onPress}
      className="gap-[5px] border-b border-neutral-900 px-[16px] py-[13px]"
      style={
        selected
          ? {
              backgroundColor: nocturne.accent[900],
              borderLeftWidth: 2,
              borderLeftColor: nocturne.accent.DEFAULT,
            }
          : undefined
      }
    >
      <View className="flex-row items-center gap-[7px]">
        <Text numberOfLines={1} className="shrink font-med text-[14px] text-fg">
          {note.title || strings.common.untitled}
        </Text>
        {note.starred ? (
          <Icon name="star" size={12} color={nocturne.accent.DEFAULT} weight="fill" />
        ) : null}
        {note.shared ? (
          <View className="ml-auto">
            <Icon name="users" size={12} color={nocturne.accent[400]} />
          </View>
        ) : null}
      </View>

      {note.preview !== "" ? (
        <Text numberOfLines={2} className="font-sans text-[12px] leading-[18px] text-neutral-500">
          {note.preview}
        </Text>
      ) : null}

      <View className="flex-row items-center gap-[7px]">
        {editorName !== "" ? <Avatar name={editorName} size={16} /> : null}
        <Text className="font-sans text-[11px] text-neutral-600">
          {editorName !== ""
            ? strings.note.editedBy(firstName(editorName), relative(note.updatedAt))
            : relative(note.updatedAt)}
        </Text>
        {note.taskTotal > 0 ? (
          <View className="ml-auto flex-row items-center gap-[4px]">
            <Icon name="check-square" size={12} color={nocturne.neutral[600]} />
            <Text className="font-sans text-[11px] text-neutral-600">
              {note.taskDone}/{note.taskTotal}
            </Text>
          </View>
        ) : null}
      </View>
    </Pressable>
  );
}

function DenseRow({ note, selected, onPress }: { note: Note; selected: boolean; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={note.title || strings.common.untitled}
      accessibilityState={{ selected }}
      onPress={onPress}
      className="h-[30px] flex-row items-center gap-[8px] px-[16px]"
      style={selected ? { backgroundColor: nocturne.accent[900] } : undefined}
    >
      <Text numberOfLines={1} className="shrink font-sans text-[13px] text-fg">
        {note.title || strings.common.untitled}
      </Text>
      {note.starred ? <Icon name="star" size={11} color={nocturne.accent.DEFAULT} weight="fill" /> : null}
      <Text className="ml-auto font-sans text-[11px] text-neutral-600">{stamp(note.updatedAt)}</Text>
    </Pressable>
  );
}

interface RecencyGroup {
  label: string;
  notes: Note[];
}

function groupByRecency(notes: readonly Note[]): RecencyGroup[] {
  const now = Date.now();
  const buckets: Record<string, Note[]> = { today: [], week: [], older: [] };
  for (const note of notes) buckets[bucket(note.updatedAt, now)]?.push(note);
  return [
    { label: strings.list.today, notes: buckets.today ?? [] },
    { label: strings.list.thisWeek, notes: buckets.week ?? [] },
    { label: strings.list.older, notes: buckets.older ?? [] },
  ].filter((group) => group.notes.length > 0);
}

export function firstName(name: string): string {
  return name.split(" ")[0] ?? name;
}

/* ------------------------------------------------------------------- mobile */

function MobileTabs() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: nocturne.accent.DEFAULT,
        tabBarInactiveTintColor: nocturne.neutral[600],
        tabBarStyle: {
          backgroundColor: nocturne.rail,
          borderTopColor: nocturne.neutral[900],
          borderTopWidth: 1,
          height: 76,
          paddingTop: 8,
          paddingBottom: 20,
          elevation: 0,
        },
        tabBarLabelStyle: { fontFamily: "Inter_500Medium", fontSize: 11 },
      }}
    >
      <Tabs.Screen name="index" options={{ title: strings.tabs.notes, tabBarIcon: tabIcon("notebook") }} />
      <Tabs.Screen
        name="search"
        options={{ title: strings.tabs.search, tabBarIcon: tabIcon("magnifying-glass") }}
      />
      <Tabs.Screen
        name="shared"
        options={{ title: strings.tabs.shared, tabBarIcon: tabIcon("users-three") }}
      />
      <Tabs.Screen
        name="settings"
        options={{ title: strings.tabs.settings, tabBarIcon: tabIcon("gear-six") }}
      />
      <Tabs.Screen name="onboarding" options={{ href: null }} />
      <Tabs.Screen name="notebook/[id]" options={{ href: null }} />
      {/* A destination, not a tab: reached from the notes header, and small enough that a
          fifth tab would cost the other four more than it is worth. */}
      <Tabs.Screen name="archive" options={{ href: null }} />
      {/* The editor is full-bleed: the block bar sits where the tab bar would be. */}
      <Tabs.Screen name="note/[id]" options={{ href: null, tabBarStyle: { display: "none" } }} />
    </Tabs>
  );
}

function tabIcon(name: IconName) {
  // React Navigation hands the tint down as ColorValue; the opaque half of that union only
  // turns up for PlatformColor, which this bar never sets.
  return function TabIcon({ color }: { color: ColorValue }) {
    return <Icon name={name} size={22} color={color as string} />;
  };
}

/**
 * The mobile list header + FAB, shared by the notes tab and a notebook screen. Drawn here
 * because both of those screens are the same screen with a different filter.
 *
 * `archiveLink` is the phone's way INTO the archive. On desktop that door is a rail row; a
 * phone has no rail, and the note menu offers Archive on both layouts — so without a door here
 * a note could be archived on a phone and then never seen again from one. It hangs off the
 * notes tab rather than off every list: the archive is a peer of "All notes", not of one
 * notebook.
 */
export function MobileListHeader({
  title,
  count,
  archiveLink = false,
}: {
  title: string;
  count: number;
  archiveLink?: boolean;
}) {
  const shell = useShell();
  const router = useRouter();
  const [sorting, setSorting] = useState(false);
  const sortLabel =
    SORT_OPTIONS.find((option) => option.sort === shell.sort)?.label ?? strings.list.sortUpdated;
  return (
    <View className="flex-row items-end gap-[8px] px-[20px] pb-[10px] pt-[6px]">
      <Text className="font-med text-[26px] text-fg" style={{ letterSpacing: -0.5 }}>
        {title}
      </Text>
      <Text className="pb-[4px] font-sans text-[12px] text-neutral-600">{count}</Text>
      <View className="ml-auto flex-row items-center gap-[6px]">
        <Chip label={sortLabel} icon="sort-ascending" onPress={() => setSorting(true)} active tone="neutral" />
        <Chip
          label={shell.density === "dense" ? strings.list.dense : strings.list.cards}
          onPress={() => shell.setDensity(shell.density === "dense" ? "cards" : "dense")}
          active
          tone="neutral"
        />
        {archiveLink ? (
          <IconButton
            icon="archive"
            label={strings.rail.archive}
            size={17}
            color={nocturne.neutral[500]}
            onPress={() => router.push("/(app)/archive")}
          />
        ) : null}
      </View>
      <SortSheet visible={sorting} onClose={() => setSorting(false)} />
    </View>
  );
}

/** The capture FAB. Mobile only — desktop captures with ⌘N. */
export function CaptureFab() {
  const shell = useShell();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={strings.capture.open}
      onPress={shell.openCapture}
      className="absolute bottom-[22px] right-[20px] h-[54px] w-[54px] items-center justify-center rounded-full border border-accent bg-accent-900"
    >
      <Icon name="plus" size={22} color={nocturne.accent[300]} />
    </Pressable>
  );
}

function Booting({ label }: { label: string }) {
  return (
    <Screen>
      <View className="flex-1 items-center justify-center gap-[10px]">
        <ActivityIndicator color={nocturne.accent.DEFAULT} />
        <Text className="font-sans text-[13px] text-neutral-500">{label}</Text>
      </View>
    </Screen>
  );
}
