import { useNotes } from "@fm/api";
import { useRouter } from "expo-router";
import { useEffect } from "react";
import { View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { EmptyState, RoundButton, Screen } from "@fm/ui";
import {
  MobileListHeader,
  NoteListBody,
  filtersFor,
  useIsDesktop,
  useShell,
} from "./_layout.tsx";

export default function ArchiveScreen() {
  const desktop = useIsDesktop();
  const shell = useShell();
  const router = useRouter();
  const notes = useNotes(filtersFor("archive", "", shell.sort));
  const rows = notes.data ?? [];

  const { select } = shell;
  useEffect(() => {
    if (desktop) select("archive");
  }, [desktop, select]);

  if (desktop) {
    return (
      <View className="flex-1 items-center justify-center bg-bg">
        <EmptyState
          title={strings.list.pickANote}
          body={strings.list.pickANoteBody}
        />
      </View>
    );
  }

  return (
    <Screen>
      <View className="flex-row items-center px-lg pt-1.5">
        <RoundButton icon="back" label={strings.note.back} onPress={() => router.back()} />
      </View>
      <MobileListHeader title={strings.rail.archive} count={rows.length} />
      <NoteListBody
        notes={rows}
        loading={notes.isPending}
        density={shell.density}
        onOpen={(id) => router.push(`/(app)/note/${id}`)}
        emptyTitle={strings.list.emptyArchiveTitle}
        emptyBody={strings.list.emptyArchiveBody}
      />
    </Screen>
  );
}
