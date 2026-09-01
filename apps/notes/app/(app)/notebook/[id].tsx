import { useNotes } from "@fm/api";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect } from "react";
import { View } from "react-native";

import { strings } from "../../../components/i18n/index.ts";
import { EmptyState, Screen } from "../../../components/nocturne/index.ts";
import { CaptureSheet } from "../../../components/offline/index.ts";
import {
  CaptureFab,
  MobileListHeader,
  NoteListBody,
  filtersFor,
  useIsDesktop,
  useShell,
} from "../_layout.tsx";

/**
 * One notebook's notes. On desktop the shell's list pane is already scoped to it — this route
 * only has to put the rail's selection where the URL says it should be, which is what makes a
 * deep link into a notebook light up the right row.
 */
export default function NotebookScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const notebookId = id ?? "";
  const desktop = useIsDesktop();
  const shell = useShell();
  const router = useRouter();
  const notes = useNotes(filtersFor("notebook", notebookId, shell.sort), {
    enabled: notebookId !== "",
  });

  const { select } = shell;
  useEffect(() => {
    if (notebookId !== "") select("notebook", notebookId);
  }, [notebookId, select]);

  const notebook = shell.notebooks.find((n) => n.id === notebookId);

  if (desktop) {
    return (
      <View className="flex-1 items-center justify-center bg-bg">
        <EmptyState
          title={strings.list.pickANote}
          body={strings.list.pickANoteBody}
          icon="folder-simple"
          action={{ label: strings.list.newNote, onPress: () => shell.newNote() }}
        />
      </View>
    );
  }

  return (
    <Screen>
      <MobileListHeader
        title={notebook?.name ?? strings.list.title}
        count={notes.data?.length ?? 0}
      />
      <NoteListBody
        notes={notes.data ?? []}
        loading={notes.isPending}
        density={shell.density}
        onOpen={(noteId) => router.push(`/(app)/note/${noteId}`)}
        emptyTitle={strings.list.emptyTitle}
        emptyBody={strings.list.emptyBody}
        onNewNote={shell.openCapture}
      />
      <CaptureFab />
      <CaptureSheet
        visible={shell.captureVisible}
        onClose={shell.closeCapture}
        notebooks={shell.notebooks}
        defaultNotebookId={notebookId}
      />
    </Screen>
  );
}
