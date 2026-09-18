import { useNoteSearch } from "@fm/api";
import { SearchFacet, type SearchHit } from "@fm/sdk/notes/v1/notes_pb";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal, Platform, Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { strings } from "./i18n/index.ts";
import { Icon, organic, type IconName } from "@fm/ui";
import { Kbd } from "./kit/index.ts";


export interface PaletteProps {
  visible: boolean;
  onClose: () => void;
  onOpenNote: (noteId: string) => void;
  onOpenNotebook: (notebookId: string) => void;
  onCreateNote: (title: string) => void;
}

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
      className={`flex-none rounded-full border px-2.25 py-0.75 ${
        active ? "border-accent bg-accent-100" : "border-transparent bg-neutral-200"
      }`}
    >
      <Text className={`font-fig text-[11.5px] ${active ? "text-accent-800" : "text-neutral-800"}`}>
        {label}
      </Text>
    </Pressable>
  );
}

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
        {
}
        <View pointerEvents="none" className="absolute inset-0" style={SCRIM} />
        <Pressable
          accessibilityRole="none"
          onPress={() => undefined}
          className="mt-24 w-[660px] max-w-[92%] overflow-hidden rounded-2xl bg-surface"
        >
          <View className="flex-row items-center gap-2.75 border-b border-neutral-300 px-4.5 py-3.75">
            <Icon name="magnifying-glass" size={18} color={organic.accent.DEFAULT} />
            <TextInput
              value={query}
              onChangeText={setQuery}
              autoFocus
              placeholder={strings.palette.placeholder}
              placeholderTextColor={organic.neutral[600]}
              accessibilityLabel={strings.palette.placeholder}
              className="flex-1 font-fig text-[17px] text-fg"
            />
            <Kbd>{strings.palette.esc}</Kbd>
          </View>

          <View className="flex-row gap-1.5 px-4.5 pb-xs pt-2.5">
            {FACETS.map((f) => (
              <FacetPill
                key={f.facet}
                label={f.label}
                active={facet === f.facet}
                onPress={() => setFacet(f.facet)}
              />
            ))}
          </View>

          <ScrollView className="max-h-[420px] px-sm pb-1.5 pt-sm">
            {query.trim() === "" ? (
              <Text className="px-2.5 py-3.5 font-fig text-[12.5px] text-neutral-700">
                {strings.palette.hint}
              </Text>
            ) : null}

            {grouped.map((group) => (
              <View key={group.label}>
                <Text className="px-2.5 pb-1.25 pt-2.5 font-fig-semi text-[10px] uppercase text-neutral-600" style={{ letterSpacing: 0.9 }}>
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
              <Text className="px-2.5 py-3.5 font-fig text-[12.5px] text-neutral-700">
                {strings.palette.empty}
              </Text>
            ) : null}

            {query.trim() !== "" ? (
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={strings.palette.createNote(query.trim())}
                onPress={create}
                className="mt-1.5 flex-row items-center gap-2.75 rounded-xl px-2.5 py-2.25"
              >
                <Icon name="plus-circle" size={16} color={organic.accent.DEFAULT} />
                <Text className="font-fig text-[13.5px] text-fg">
                  {strings.palette.createNote(query.trim())}
                </Text>
                <View className="ml-auto">
                  <Kbd>⌘⏎</Kbd>
                </View>
              </Pressable>
            ) : null}
          </ScrollView>

          <View className="flex-row items-center gap-lg border-t border-neutral-300 px-4.5 py-2.25">
            <FooterHint keys="↑↓" label={strings.palette.navigate} />
            <FooterHint keys="⇥" label={strings.palette.filter} />
            <FooterHint keys="⌘⏎" label={strings.palette.newNote} />
            <Text className="ml-auto font-fig text-[11px] text-neutral-700">
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
    <View className="flex-row items-center gap-1.25">
      <Kbd>{keys}</Kbd>
      <Text className="font-fig text-[11px] text-neutral-700">{label}</Text>
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
      className={`flex-row items-center gap-2.75 rounded-xl px-2.5 py-2.25 ${
        selected ? "bg-accent-100" : ""
      }`}
    >
      <Icon
        name={GLYPH[hit.kind] ?? "file-text"}
        size={16}
        color={selected ? organic.accent[700] : organic.neutral[700]}
      />
      <View className="min-w-0 flex-1 gap-0.5">
        <Highlighted
          text={hit.title}
          query={query}
          className="font-fig-med text-[13.5px] text-fg"
        />
        {hit.context !== "" || hit.snippet !== "" ? (
          <Text numberOfLines={1} className="font-fig text-[11.5px] text-neutral-700">
            {[hit.context, hit.snippet].filter((part) => part !== "").join(" · ")}
          </Text>
        ) : null}
      </View>
      {selected ? <Kbd>↵</Kbd> : null}
    </Pressable>
  );
}

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
      <Text style={{ backgroundColor: organic.accent[200] }}>{text.slice(at, at + term.length)}</Text>
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

const SCRIM = { backgroundColor: organic.bg, opacity: 0.62 } as const;
