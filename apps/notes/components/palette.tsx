import { useNoteSearch } from "@fm/api";
import { SearchFacet, type SearchHit } from "@fm/sdk/notes/v1/notes_pb";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal, Platform, Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { strings } from "./i18n/index.ts";
import { Icon, Kbd, nocturne, type IconName } from "./nocturne/index.ts";

/**
 * The ⌘K palette (artboard 1d). Desktop only.
 *
 * Search is the SERVER's, not a filter over a loaded list: the proto is explicit about why —
 * a client-side filter stops working at the first notebook nobody has opened yet. So this
 * component owns a query string and a facet, and everything else is drawn from the response,
 * including the footer's "Searched 128 notes in 31ms", which is the server's own measurement
 * rather than a round trip the user's network dominates.
 *
 * The key handling is a raw `window` listener behind a Platform guard. React Native has no
 * key events for a component that is not a text input, and the palette's ↑↓/↵/⌘↵/esc have to
 * work while the caret is in the field.
 */

export interface PaletteProps {
  visible: boolean;
  onClose: () => void;
  onOpenNote: (noteId: string) => void;
  onOpenNotebook: (notebookId: string) => void;
  onCreateNote: (title: string) => void;
}

/**
 * The facet row, shared by the ⌘K palette and the mobile search tab so the same control keeps
 * one shape. 1d draws these as fully-round pills, which the shared Chip (rounded-md, 8px)
 * cannot be talked into without editing it; this is that pill, in one place instead of two.
 */
export function FacetPill({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected: active }}
      onPress={onPress}
      className={`flex-none rounded-full border px-[9px] py-[3px] ${
        active ? "border-accent bg-accent-900" : "border-transparent bg-neutral-900"
      }`}
    >
      <Text className={`font-sans text-[11.5px] ${active ? "text-accent-300" : "text-neutral-400"}`}>
        {label}
      </Text>
    </Pressable>
  );
}

/** The facets, in the order ⇥ cycles them and the order the search tab draws them. */
export const FACETS: readonly { facet: SearchFacet; label: string }[] = [
  { facet: SearchFacet.ALL, label: strings.palette.facetAll },
  { facet: SearchFacet.NOTES, label: strings.palette.facetNotes },
  { facet: SearchFacet.TASKS, label: strings.palette.facetTasks },
  { facet: SearchFacet.NOTEBOOKS, label: strings.palette.facetNotebooks },
];

export function Palette({ visible, onClose, onOpenNote, onOpenNotebook, onCreateNote }: PaletteProps) {
  const [query, setQuery] = useState("");
  const [facet, setFacet] = useState<SearchFacet>(SearchFacet.ALL);
  const [cursor, setCursor] = useState(0);

  const search = useNoteSearch(query, facet, { limit: 20, enabled: visible });
  const hits = useMemo(() => search.data?.hits ?? [], [search.data]);

  useEffect(() => {
    setCursor(0);
  }, [query, facet]);

  useEffect(() => {
    if (!visible) {
      setQuery("");
      setFacet(SearchFacet.ALL);
    }
  }, [visible]);

  const open = useCallback(
    (hit: SearchHit) => {
      if (hit.kind === SearchFacet.NOTEBOOKS && hit.notebookId !== "") onOpenNotebook(hit.notebookId);
      else if (hit.noteId !== "") onOpenNote(hit.noteId);
      onClose();
    },
    [onClose, onOpenNote, onOpenNotebook],
  );

  const create = useCallback(() => {
    const title = query.trim();
    if (title === "") return;
    onCreateNote(title);
    onClose();
  }, [onClose, onCreateNote, query]);

  useEffect(() => {
    if (Platform.OS !== "web" || !visible) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
        return;
      }
      if (event.key === "Enter") {
        event.preventDefault();
        if (event.metaKey || event.ctrlKey) create();
        else {
          const hit = hits[cursor];
          if (hit) open(hit);
        }
        return;
      }
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setCursor((c) => (hits.length === 0 ? 0 : Math.min(c + 1, hits.length - 1)));
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setCursor((c) => Math.max(0, c - 1));
        return;
      }
      if (event.key === "Tab") {
        event.preventDefault();
        setFacet((current) => {
          const index = FACETS.findIndex((f) => f.facet === current);
          return FACETS[(index + 1) % FACETS.length]?.facet ?? SearchFacet.ALL;
        });
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [visible, hits, cursor, onClose, open, create]);

  const grouped = useMemo(() => groupHits(hits), [hits]);
  let flatIndex = -1;

  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.common.close}
        onPress={onClose}
        className="flex-1 items-center"
      >
        {/* The scrim is its own layer: putting the opacity on the parent would fade the
            palette with it. */}
        <View pointerEvents="none" className="absolute inset-0" style={SCRIM} />
        <Pressable
          accessibilityRole="none"
          onPress={() => undefined}
          className="mt-[96px] w-[660px] max-w-[92%] overflow-hidden rounded-lg bg-surface"
        >
          <View className="flex-row items-center gap-[11px] border-b border-neutral-800 px-[18px] py-[15px]">
            <Icon name="magnifying-glass" size={18} color={nocturne.accent.DEFAULT} />
            <TextInput
              value={query}
              onChangeText={setQuery}
              autoFocus
              placeholder={strings.palette.placeholder}
              placeholderTextColor={nocturne.neutral[600]}
              accessibilityLabel={strings.palette.placeholder}
              className="flex-1 font-sans text-[17px] text-fg"
            />
            <Kbd>{strings.palette.esc}</Kbd>
          </View>

          <View className="flex-row gap-[6px] px-[18px] pb-[4px] pt-[10px]">
            {FACETS.map((f) => (
              <FacetPill
                key={f.facet}
                label={f.label}
                active={facet === f.facet}
                onPress={() => setFacet(f.facet)}
              />
            ))}
          </View>

          <ScrollView className="max-h-[420px] px-[8px] pb-[6px] pt-[8px]">
            {query.trim() === "" ? (
              <Text className="px-[10px] py-[14px] font-sans text-[12.5px] text-neutral-500">
                {strings.palette.hint}
              </Text>
            ) : null}

            {grouped.map((group) => (
              <View key={group.label}>
                <Text className="px-[10px] pb-[5px] pt-[10px] font-semi text-[10px] uppercase text-neutral-600" style={{ letterSpacing: 0.9 }}>
                  {group.label}
                </Text>
                {group.hits.map((hit) => {
                  flatIndex += 1;
                  const index = flatIndex;
                  return (
                    <HitRow
                      key={`${hit.kind}-${hit.noteId}-${hit.notebookId}-${index}`}
                      hit={hit}
                      query={query}
                      selected={index === cursor}
                      onPress={() => open(hit)}
                    />
                  );
                })}
              </View>
            ))}

            {query.trim() !== "" && hits.length === 0 && !search.isPending ? (
              <Text className="px-[10px] py-[14px] font-sans text-[12.5px] text-neutral-500">
                {strings.palette.empty}
              </Text>
            ) : null}

            {query.trim() !== "" ? (
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={strings.palette.createNote(query.trim())}
                onPress={create}
                className="mt-[6px] flex-row items-center gap-[11px] rounded-md px-[10px] py-[9px]"
              >
                <Icon name="plus-circle" size={16} color={nocturne.accent.DEFAULT} />
                <Text className="font-sans text-[13.5px] text-fg">
                  {strings.palette.createNote(query.trim())}
                </Text>
                <View className="ml-auto">
                  <Kbd>⌘⏎</Kbd>
                </View>
              </Pressable>
            ) : null}
          </ScrollView>

          <View className="flex-row items-center gap-[16px] border-t border-neutral-800 px-[18px] py-[9px]">
            <FooterHint keys="↑↓" label={strings.palette.navigate} />
            <FooterHint keys="⇥" label={strings.palette.filter} />
            <FooterHint keys="⌘⏎" label={strings.palette.newNote} />
            <Text className="ml-auto font-sans text-[11px] text-neutral-500">
              {search.data
                ? strings.palette.footer(search.data.searchedNotes, search.data.elapsedMs)
                : ""}
            </Text>
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

function FooterHint({ keys, label }: { keys: string; label: string }) {
  return (
    <View className="flex-row items-center gap-[5px]">
      <Kbd>{keys}</Kbd>
      <Text className="font-sans text-[11px] text-neutral-500">{label}</Text>
    </View>
  );
}

function HitRow({
  hit,
  query,
  selected,
  onPress,
}: {
  hit: SearchHit;
  query: string;
  selected: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={hit.title}
      accessibilityState={{ selected }}
      onPress={onPress}
      className={`flex-row items-center gap-[11px] rounded-md px-[10px] py-[9px] ${
        selected ? "bg-accent-900" : ""
      }`}
    >
      <Icon
        name={GLYPH[hit.kind] ?? "file-text"}
        size={16}
        color={selected ? nocturne.accent[300] : nocturne.neutral[500]}
      />
      <View className="min-w-0 flex-1 gap-[2px]">
        <Highlighted
          text={hit.title}
          query={query}
          className="font-med text-[13.5px] text-fg"
        />
        {hit.context !== "" || hit.snippet !== "" ? (
          <Text numberOfLines={1} className="font-sans text-[11.5px] text-neutral-500">
            {[hit.context, hit.snippet].filter((part) => part !== "").join(" · ")}
          </Text>
        ) : null}
      </View>
      {selected ? <Kbd>↵</Kbd> : null}
    </Pressable>
  );
}

/**
 * The server sends the matching text with the terms left in place and no markup — the proto
 * says so — so the highlight is drawn here, by splitting on the query.
 */
function Highlighted({ text, query, className }: { text: string; query: string; className: string }) {
  const term = query.trim();
  const at = term === "" ? -1 : text.toLowerCase().indexOf(term.toLowerCase());
  if (at < 0) {
    return (
      <Text numberOfLines={1} className={className}>
        {text}
      </Text>
    );
  }
  return (
    <Text numberOfLines={1} className={className}>
      {text.slice(0, at)}
      <Text style={{ backgroundColor: nocturne.accent[800] }}>{text.slice(at, at + term.length)}</Text>
      {text.slice(at + term.length)}
    </Text>
  );
}

const GLYPH: Partial<Record<SearchFacet, IconName>> = {
  [SearchFacet.NOTES]: "file-text",
  [SearchFacet.TASKS]: "check-square",
  [SearchFacet.NOTEBOOKS]: "folder-simple",
};

interface HitGroup {
  label: string;
  hits: SearchHit[];
}

function groupHits(hits: readonly SearchHit[]): HitGroup[] {
  const order: { kind: SearchFacet; label: string }[] = [
    { kind: SearchFacet.NOTES, label: strings.palette.groupNotes },
    { kind: SearchFacet.TASKS, label: strings.palette.groupTasks },
    { kind: SearchFacet.NOTEBOOKS, label: strings.palette.groupNotebooks },
  ];
  const groups: HitGroup[] = [];
  for (const { kind, label } of order) {
    const found = hits.filter((hit) => hit.kind === kind);
    if (found.length > 0) groups.push({ label, hits: found });
  }
  const rest = hits.filter(
    (hit) => hit.kind !== SearchFacet.NOTES && hit.kind !== SearchFacet.TASKS && hit.kind !== SearchFacet.NOTEBOOKS,
  );
  if (rest.length > 0) groups.push({ label: strings.palette.groupNotes, hits: rest });
  return groups;
}

/**
 * Binds ⌘K and ⌘N on web. A no-op everywhere else — the shortcuts are the desktop shell's,
 * and a phone has no meta key to press.
 */
export function useDesktopShortcuts({
  onPalette,
  onNewNote,
}: {
  onPalette: () => void;
  onNewNote: () => void;
}) {
  useEffect(() => {
    if (Platform.OS !== "web") return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (!(event.metaKey || event.ctrlKey)) return;
      const key = event.key.toLowerCase();
      if (key === "k") {
        event.preventDefault();
        onPalette();
      } else if (key === "n") {
        event.preventDefault();
        onNewNote();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onPalette, onNewNote]);
}

/**
 * The overlay the design dims the app with behind a sheet. Drawn as the ground colour at
 * opacity rather than as a fourth hard-coded rgba: the ground is a token, and the scrim is
 * "the ground, mostly opaque".
 */
const SCRIM = { backgroundColor: nocturne.bg, opacity: 0.62 } as const;
