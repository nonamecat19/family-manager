import { useNoteSearch } from "@fm/api";
import { SearchFacet, type SearchHit } from "@fm/sdk/notes/v1/notes_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { Icon, Screen, nocturne, type IconName } from "../../components/nocturne/index.ts";
import { FACETS, FacetPill } from "../../components/palette.tsx";
import { useShell } from "./_layout.tsx";

export default function SearchScreen() {
  const router = useRouter();
  const shell = useShell();
  const [query, setQuery] = useState("");
  const [facet, setFacet] = useState<SearchFacet>(SearchFacet.ALL);
  const search = useNoteSearch(query, facet, { limit: 40 });
  const hits = search.data?.hits ?? [];

  const open = (hit: SearchHit) => {
    if (hit.kind === SearchFacet.NOTEBOOKS && hit.notebookId !== "") {
      shell.select("notebook", hit.notebookId);
      router.push(`/(app)/notebook/${hit.notebookId}`);
      return;
    }
    if (hit.noteId !== "") router.push(`/(app)/note/${hit.noteId}`);
  };

  return (
    <Screen>
      <View className="gap-[12px] px-[20px] pb-[10px] pt-[6px]">
        <Text className="font-med text-[26px] text-fg" style={{ letterSpacing: -0.5 }}>
          {strings.search.title}
        </Text>
        <View className="h-[38px] flex-row items-center gap-[9px] rounded-md border border-neutral-800 bg-surface px-[11px]">
          <Icon name="magnifying-glass" size={16} color={nocturne.accent.DEFAULT} />
          <TextInput
            value={query}
            onChangeText={setQuery}
            placeholder={strings.search.placeholder}
            placeholderTextColor={nocturne.neutral[600]}
            accessibilityLabel={strings.search.placeholder}
            className="flex-1 font-sans text-[14px] text-fg"
          />
        </View>
        <View className="flex-row gap-[7px]">
          {FACETS.map((f) => (
            <FacetPill
              key={f.facet}
              label={f.label}
              active={facet === f.facet}
              onPress={() => setFacet(f.facet)}
            />
          ))}
        </View>
      </View>

      <ScrollView className="px-[12px]">
        {query.trim() === "" ? (
          <Text className="px-[8px] py-[14px] font-sans text-[13px] text-neutral-500">
            {strings.search.hint}
          </Text>
        ) : null}
        {query.trim() !== "" && hits.length === 0 && !search.isPending ? (
          <Text className="px-[8px] py-[14px] font-sans text-[13px] text-neutral-500">
            {strings.search.empty}
          </Text>
        ) : null}
        {hits.map((hit, index) => (
          <Pressable
            key={`${hit.kind}-${hit.noteId}-${hit.notebookId}-${index}`}
            accessibilityRole="button"
            accessibilityLabel={hit.title}
            onPress={() => open(hit)}
            className="flex-row items-center gap-[11px] rounded-md px-[8px] py-[11px]"
          >
            <Icon name={GLYPH[hit.kind] ?? "file-text"} size={17} color={nocturne.neutral[500]} />
            <View className="min-w-0 flex-1 gap-[2px]">
              <Text numberOfLines={1} className="font-med text-[14px] text-fg">
                {hit.title}
              </Text>
              {hit.context !== "" || hit.snippet !== "" ? (
                <Text numberOfLines={1} className="font-sans text-[12px] text-neutral-500">
                  {[hit.context, hit.snippet].filter((part) => part !== "").join(" · ")}
                </Text>
              ) : null}
            </View>
          </Pressable>
        ))}
        {search.data ? (
          <Text className="px-[8px] py-[14px] font-sans text-[11px] text-neutral-600">
            {strings.palette.footer(search.data.searchedNotes, search.data.elapsedMs)}
          </Text>
        ) : null}
      </ScrollView>
    </Screen>
  );
}

const GLYPH: Partial<Record<SearchFacet, IconName>> = {
  [SearchFacet.NOTES]: "file-text",
  [SearchFacet.TASKS]: "check-square",
  [SearchFacet.NOTEBOOKS]: "folder-simple",
};
