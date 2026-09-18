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
} from "react-native";

import { bucket, relative, stamp, strings } from "../../components/i18n/index.ts";
import {
  Avatar,
  BootSplash,
  Chip,
  Divider,
  EmptyState,
  GateMessage,
  Icon,
  IconButton,
  PrimaryButton,
  Screen,
  organic,
  tabIcon,
  useTabScreenOptions,
  type IconName,
} from "@fm/ui";
import { AvatarStack, Kbd, Pane, Rail } from "../../components/kit/index.ts";
import { Palette, useDesktopShortcuts } from "../../components/palette.tsx";
import { useMemberDirectory } from "../../components/share.tsx";
import { makeBlock } from "../../components/editor/index.ts";
import type { QueuedNote } from "../../components/offline/index.ts";


export const DESKTOP_MIN_WIDTH = 1024;

export function useIsDesktop(): boolean {
  const { width } = useWindowDimensions();
  return width >= DESKTOP_MIN_WIDTH;
}

export const PANE = {
  rail: 236,
  list: 322,
  editorRail: 268,
  comments: 200,
  minColumn: 460,
} as const;

export interface PaneFit {
  desktop: boolean;
  list: boolean;
  editorRail: boolean;
  comments: boolean;
}

export function usePaneFit(): PaneFit {
  const { width } = useWindowDimensions();
  const pathname = usePathname();

  if (width < DESKTOP_MIN_WIDTH) {
    return { desktop: false, list: false, editorRail: false, comments: false };
  }
  if (!pathname.startsWith("/note/")) {
    return { desktop: true, list: true, editorRail: false, comments: false };
  }

  let spare = width - PANE.rail - PANE.minColumn;
  const editorRail = spare >= PANE.editorRail;
  if (editorRail) spare -= PANE.editorRail;
  const list = spare >= PANE.list;
  if (list) spare -= PANE.list;
  return { desktop: true, list, editorRail, comments: list && spare >= PANE.comments };
}

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

export const SORT_OPTIONS: readonly { sort: NoteSort; label: string }[] = [
  { sort: NoteSort.UPDATED, label: strings.list.sortUpdated },
  { sort: NoteSort.CREATED, label: strings.list.sortCreated },
  { sort: NoteSort.TITLE, label: strings.list.sortTitle },
];

export default function AppLayout() {
  return <Gate />;
}

function Gate() {
  const family = useFamily();
  const router = useRouter();
  const segments = useSegments();
  const isOnboarding = segments[segments.length - 1] === "onboarding";

  if (family.isPending) return <BootSplash title={strings.app.name} label={strings.gate.settingUp} />;

  if (family.isError) {
    const code = (family.error as { code?: Code }).code;
    if (code === Code.FailedPrecondition) {
      if (isOnboarding) return <Slot />;
      return (
        <Screen>
          <GateMessage
            title={strings.gate.noFamilyTitle}
            body={strings.gate.noFamilyBody}
            actionTitle={strings.gate.createFamily}
            onAction={() => router.push("/(app)/onboarding")}
          />
        </Screen>
      );
    }
    const shown = toDisplayError(family.error, strings.common.loadFailed);
    return (
      <Screen>
        <GateMessage
          title={strings.gate.errorTitle}
          body={shown.message}
          reference={shown.reference ? strings.common.errorReference(shown.reference) : undefined}
          actionTitle={strings.common.tryAgain}
          onAction={() => void family.refetch()}
        />
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
  const [showArchivedNotebooks, setShowArchivedNotebooks] = useState(false);

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

  const tree = useMemo(() => buildTree(shell.notebooks), [shell.notebooks]);

  const failed = createNotebook.isError || updateNotebook.isError || deleteNotebook.isError;

  return (
    <Rail
      className="flex-none gap-3.5 px-3 py-lg"
      style={{ width: PANE.rail }}
    >
      <View className="flex-row items-center gap-2.25 px-xs">
        <View className="h-[22px] w-[22px] items-center justify-center rounded-xl border border-accent">
          <Text className="font-fig-semi text-[11px] text-accent-700">C</Text>
        </View>
        <Text className="font-fig-med text-[15px] text-fg">{strings.app.name}</Text>
      </View>

      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.rail.search}
        onPress={shell.openPalette}
        className="h-[32px] flex-row items-center gap-sm rounded-xl border border-neutral-300 bg-surface px-2.5"
      >
        <Icon name="magnifying-glass" size={14} color={organic.neutral[700]} />
        <Text className="font-fig text-[13px] text-neutral-700">{strings.rail.search}</Text>
        <View className="ml-auto">
          <Kbd>⌘K</Kbd>
        </View>
      </Pressable>

      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.rail.newNote}
        onPress={() => shell.newNote()}
        className="h-[34px] flex-row items-center justify-center gap-1.75 rounded-xl border border-accent"
      >
        <Icon name="plus" size={14} color={organic.accent.DEFAULT} />
        <Text className="font-fig-med text-[13px] text-accent-700">{strings.rail.newNote}</Text>
        <Kbd>⌘N</Kbd>
      </Pressable>

      <View className="gap-0.25">
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

      <View className="my-0.5">
        <Divider />
      </View>

      <View className="flex-row items-center px-2.5">
        <Text className="font-fig-semi text-[10px] uppercase text-neutral-700" style={{ letterSpacing: 1 }}>
          {strings.rail.notebooks}
        </Text>
        <View className="ml-auto">
          <MiniButton icon="plus" label={strings.rail.newNotebook} size={13} onPress={() => setCreating(true)} />
        </View>
      </View>

      <ScrollView className="grow-0" showsVerticalScrollIndicator={false}>
        <View className="gap-0.25">
          {tree.map((node) => (
            <View key={node.notebook.id}>
              <RailRow
                icon="folder-simple"
                iconColor={organic.accent[600]}
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
        <Text className="px-2.5 font-fig text-[11px] text-neutral-700">
          {strings.rail.notebookFailed}
        </Text>
      ) : null}

      <NotebookDialog
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

      <View className="mt-auto flex-row items-center gap-sm px-2.5 pt-sm">
        <AvatarStack names={members.map((m) => m.displayName || m.email)} size={22} max={3} />
        <Text className="font-fig text-[11px] text-neutral-700">
          {strings.rail.familyCount(members.length)}
        </Text>
        <View className="ml-auto">
          <IconButton
            icon="gear-six"
            label={strings.rail.settings}
            size={15}
            color={organic.neutral[700]}
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
  onMenu?: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected: active }}
      onPress={onPress}
      onLongPress={onMenu}
      className={`h-[30px] flex-row items-center gap-2.25 rounded-xl ${nested ? "pl-5.5 pr-2.5" : "px-2.5"}`}
      style={active ? { backgroundColor: organic.accent[100] } : undefined}
    >
      {icon ? (
        <Icon
          name={icon}
          size={15}
          color={active ? organic.accent[700] : (iconColor ?? organic.neutral[700])}
          weight={active ? "fill" : "regular"}
        />
      ) : nested ? (
        <View className="h-1.25 w-1.25 rounded-full bg-neutral-600" />
      ) : null}
      <Text
        numberOfLines={1}
        className={`shrink font-fig text-[13px] ${active ? "text-accent-800" : "text-neutral-700"}`}
      >
        {label}
      </Text>
      {typeof count === "number" && count > 0 ? (
        <Text className="ml-auto font-fig text-[11px] text-neutral-600">{count}</Text>
      ) : null}
      {onMenu ? (
        <View className={typeof count === "number" && count > 0 ? "" : "ml-auto"}>
          <MiniButton icon="dots-three" label={strings.rail.notebookActions} size={14} onPress={onMenu} />
        </View>
      ) : null}
    </Pressable>
  );
}

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
      className="h-[20px] w-[20px] items-center justify-center rounded-md"
    >
      <Icon name={icon} size={size} color={organic.neutral[600]} />
    </Pressable>
  );
}


const SCRIM = { backgroundColor: organic.text, opacity: 0.35 } as const;

export interface SheetAction {
  key: string;
  label: string;
  icon?: IconName;
  selected?: boolean;
  tone?: "default" | "danger";
  onPress: () => void;
}

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
      <View className="flex-1 items-center justify-center px-xl" pointerEvents="box-none">
        <View className="w-full max-w-[320px] gap-1.5 rounded-2xl bg-surface px-2.5 py-3 shadow-card">
          <Text className="px-sm pb-0.5 font-fig text-[12px] leading-[18px] text-neutral-700">
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
              className="h-[36px] flex-row items-center gap-2.5 rounded-xl px-sm"
            >
              {action.icon ? (
                <Icon
                  name={action.icon}
                  size={15}
                  color={action.tone === "danger" ? organic.danger : organic.accent[600]}
                />
              ) : null}
              <Text
                className={`font-fig text-[13.5px] ${
                  action.tone === "danger" ? "text-error" : "text-fg"
                }`}
              >
                {action.label}
              </Text>
              {action.selected ? (
                <View className="ml-auto">
                  <Icon name="check" size={13} color={organic.accent.DEFAULT} />
                </View>
              ) : null}
            </Pressable>
          ))}
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={strings.common.cancel}
            onPress={onClose}
            className="h-[34px] items-center justify-center rounded-xl"
          >
            <Text className="font-fig text-[13px] text-neutral-700">{strings.common.cancel}</Text>
          </Pressable>
        </View>
      </View>
    </Modal>
  );
}

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
      <View className="flex-1 items-center justify-center px-xl" pointerEvents="box-none">
        <View className="w-full max-w-[320px] gap-3 rounded-2xl bg-surface p-lg shadow-card">
          <Text className="font-fig-med text-[15px] text-fg">{title}</Text>
          <TextInput
            value={name}
            onChangeText={setName}
            autoFocus
            placeholder={strings.rail.notebookNamePlaceholder}
            placeholderTextColor={organic.neutral[600]}
            accessibilityLabel={strings.rail.notebookName}
            onSubmitEditing={() => {
              if (name.trim() !== "") onSubmit(name.trim());
            }}
            className="rounded-xl border border-neutral-300 px-3 py-2.25 font-fig text-[14px] text-fg"
          />
          <View className="flex-row items-center gap-sm">
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={strings.common.cancel}
              onPress={onClose}
              className="rounded-xl border border-neutral-300 px-3.5 py-2.25"
            >
              <Text className="font-fig-med text-[13px] text-neutral-900">{strings.common.cancel}</Text>
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

function buildTree(notebooks: readonly Notebook[]): TreeNode[] {
  const roots = notebooks.filter((n) => n.parentId === "");
  return roots.map((notebook) => ({
    notebook,
    children: notebooks.filter((n) => n.parentId === notebook.id),
  }));
}

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
      className="flex-none border-x border-neutral-300"
      style={{ width: PANE.list }}
    >
      <View className="flex-row items-center gap-sm px-lg pb-3 pt-lg">
        <Text className="font-fig-med text-[17px] text-fg">{title}</Text>
        <Text className="font-fig text-[11px] text-neutral-600">
          {strings.list.noteCount(rows.length)}
        </Text>
        <View className="ml-auto flex-row items-center gap-0.5">
          <SortButton />
          {
}
          <IconButton
            icon={shell.density === "dense" ? "list-bullets" : "rows"}
            label={strings.list.density}
            size={15}
            color={organic.neutral[700]}
            onPress={() => shell.setDensity(shell.density === "dense" ? "cards" : "dense")}
          />
          <IconButton
            icon="funnel-simple"
            label={strings.list.filter}
            size={15}
            color={organic.neutral[700]}
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

function SortButton() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <IconButton
        icon="sort-ascending"
        label={strings.list.sort}
        size={15}
        color={organic.neutral[700]}
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


export interface NoteListBodyProps {
  notes: readonly Note[];
  loading: boolean;
  density: "cards" | "dense";
  selectedId?: string;
  onOpen: (noteId: string) => void;
  emptyTitle: string;
  emptyBody: string;
  onNewNote?: () => void;
  queued?: readonly QueuedNote[];
}

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
      <View className="items-center py-10">
        <ActivityIndicator color={organic.accent.DEFAULT} />
      </View>
    );
  }

  if (notes.length === 0 && queued.length === 0) {
    return (
      <EmptyState
        title={emptyTitle}
        body={emptyBody}
        action={onNewNote ? { label: strings.list.newNote, onPress: onNewNote } : undefined}
      />
    );
  }

  const groups = groupByRecency(notes);

  return (
    <ScrollView showsVerticalScrollIndicator={false}>
      {queued.map((item) => (
        <View key={item.clientId} className="border-b border-neutral-300 px-lg py-3.25">
          <View className="flex-row items-center gap-1.75">
            <Text className="font-fig-med text-[14px] text-fg" numberOfLines={1}>
              {item.title || strings.common.untitled}
            </Text>
            <View className="ml-auto h-[7px] w-[7px] rounded-full bg-neutral-600" />
          </View>
          <Text className="pt-xs font-fig text-[11px] text-neutral-600">
            {strings.list.notSynced}
          </Text>
        </View>
      ))}

      {groups.map((group) => (
        <View key={group.label}>
          {density === "dense" ? (
            <Text className="px-lg pb-xs pt-3.5 font-fig-semi text-[10px] uppercase text-neutral-600" style={{ letterSpacing: 1 }}>
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
      className="gap-1.25 border-b border-neutral-300 px-lg py-3.25"
      style={
        selected
          ? {
              backgroundColor: organic.accent[100],
              borderLeftWidth: 2,
              borderLeftColor: organic.accent.DEFAULT,
            }
          : undefined
      }
    >
      <View className="flex-row items-center gap-1.75">
        <Text numberOfLines={1} className="shrink font-fig-med text-[14px] text-fg">
          {note.title || strings.common.untitled}
        </Text>
        {note.starred ? (
          <Icon name="star" size={12} color={organic.accent.DEFAULT} weight="fill" />
        ) : null}
        {note.shared ? (
          <View className="ml-auto">
            <Icon name="users" size={12} color={organic.accent[600]} />
          </View>
        ) : null}
      </View>

      {note.preview !== "" ? (
        <Text numberOfLines={2} className="font-fig text-[12px] leading-[18px] text-neutral-700">
          {note.preview}
        </Text>
      ) : null}

      <View className="flex-row items-center gap-1.75">
        {editorName !== "" ? <Avatar name={editorName} size={16} /> : null}
        <Text className="font-fig text-[11px] text-neutral-600">
          {editorName !== ""
            ? strings.note.editedBy(firstName(editorName), relative(note.updatedAt))
            : relative(note.updatedAt)}
        </Text>
        {note.taskTotal > 0 ? (
          <View className="ml-auto flex-row items-center gap-xs">
            <Icon name="check-square" size={12} color={organic.neutral[600]} />
            <Text className="font-fig text-[11px] text-neutral-600">
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
      className="h-[30px] flex-row items-center gap-sm px-lg"
      style={selected ? { backgroundColor: organic.accent[100] } : undefined}
    >
      <Text numberOfLines={1} className="shrink font-fig text-[13px] text-fg">
        {note.title || strings.common.untitled}
      </Text>
      {note.starred ? <Icon name="star" size={11} color={organic.accent.DEFAULT} weight="fill" /> : null}
      <Text className="ml-auto font-fig text-[11px] text-neutral-600">{stamp(note.updatedAt)}</Text>
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


function MobileTabs() {
  const tabScreenOptions = useTabScreenOptions();
  return (
    <Tabs screenOptions={tabScreenOptions}>
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
      <Tabs.Screen name="connected-accounts" options={{ href: null }} />
      <Tabs.Screen name="approve-device" options={{ href: null }} />
      <Tabs.Screen name="notebook/[id]" options={{ href: null }} />
      {
}
      <Tabs.Screen name="archive" options={{ href: null }} />
      {}
      <Tabs.Screen name="note/[id]" options={{ href: null, tabBarStyle: { display: "none" } }} />
    </Tabs>
  );
}

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
    <View className="flex-row items-end gap-sm px-5 pb-2.5 pt-1.5">
      <Text
        numberOfLines={1}
        className="flex-1 font-cap text-[26px] text-fg"
        style={{ letterSpacing: -0.5 }}
      >
        {title}
      </Text>
      <Text className="pb-xs font-fig text-[12px] text-neutral-600">{count}</Text>
      <View className="ml-auto flex-row items-center gap-1.5">
        <Chip label={sortLabel} onPress={() => setSorting(true)} active tone="accent2" />
        <Chip
          label={shell.density === "dense" ? strings.list.dense : strings.list.cards}
          onPress={() => shell.setDensity(shell.density === "dense" ? "cards" : "dense")}
          active
          tone="accent2"
        />
        {archiveLink ? (
          <IconButton
            icon="archive"
            label={strings.rail.archive}
            size={17}
            color={organic.neutral[700]}
            onPress={() => router.push("/(app)/archive")}
          />
        ) : null}
      </View>
      <SortSheet visible={sorting} onClose={() => setSorting(false)} />
    </View>
  );
}

export function CaptureFab() {
  const shell = useShell();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={strings.capture.open}
      onPress={shell.openCapture}
      className="absolute bottom-[22px] right-[20px] h-[54px] w-[54px] items-center justify-center rounded-full bg-accent shadow-fab"
    >
      <Icon name="plus" size={22} color={organic.accentFg} />
    </Pressable>
  );
}
