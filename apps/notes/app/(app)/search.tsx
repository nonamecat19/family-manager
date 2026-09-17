import { useNoteSearch } from "@fm/api";
import { SearchFacet, type SearchHit } from "@fm/sdk/notes/v1/notes_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { Icon, Screen, organic, type IconName } from "@fm/ui";
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
      <View className="gap-3 px-5 pb-2.5 pt-1.5">
        <Text className="font-cap text-26 text-fg" style={{ letterSpacing: -0.5 }}>
          {strings.search.title}
        </Text>
        <View className="h-[38px] flex-row items-center gap-2.25 rounded-xl border border-neutral-300 bg-surface px-2.75">
          <Icon name="magnifying-glass" size={16} color={organic.accent.DEFAULT} />
          <TextInput
            value={query}
            onChangeText={setQuery}
            placeholder={strings.search.placeholder}
            placeholderTextColor={organic.neutral[600]}
            accessibilityLabel={strings.search.placeholder}
            className="flex-1 font-fig text-14 text-fg"
          />
        </View>
        <View className="flex-row gap-1.75">
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

      <ScrollView className="px-3">
        {query.trim() === "" ? (
          <Text className="px-sm py-3.5 font-fig text-13 text-neutral-700">
            {strings.search.hint}
          </Text>
        ) : null}
        {query.trim() !== "" && hits.length === 0 && !search.isPending ? (
          <Text className="px-sm py-3.5 font-fig text-13 text-neutral-700">
            {strings.search.empty}
          </Text>
        ) : null}
        {hits.map((hit, index) => (
          <Pressable
            key={`${hit.kind}-${hit.noteId}-${hit.notebookId}-${index}`}
            accessibilityRole="button"
            accessibilityLabel={hit.title}
            onPress={() => open(hit)}
            className="flex-row items-center gap-2.75 rounded-xl px-sm py-2.75"
          >
            <Icon name={GLYPH[hit.kind] ?? "file-text"} size={17} color={organic.neutral[700]} />
            <View className="min-w-0 flex-1 gap-0.5">
              <Text numberOfLines={1} className="font-fig-med text-14 text-fg">
                {hit.title}
              </Text>
              {hit.context !== "" || hit.snippet !== "" ? (
                <Text numberOfLines={1} className="font-fig text-12 text-neutral-700">
                  {[hit.context, hit.snippet].filter((part) => part !== "").join(" · ")}
                </Text>
              ) : null}
            </View>
          </Pressable>
        ))}
        {search.data ? (
          <Text className="px-sm py-3.5 font-fig text-11 text-neutral-600">
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
