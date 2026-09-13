import { useNotes } from "@fm/api";
import { useRouter } from "expo-router";
import { Text, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { CaptureSheet, useCaptureQueue } from "../../components/offline/index.ts";
import { EmptyState, Screen } from "../../components/nocturne/index.ts";
import {
  CaptureFab,
  MobileListHeader,
  NoteListBody,
  filtersFor,
  useIsDesktop,
  useShell,
} from "./_layout.tsx";

export default function NotesScreen() {
  const desktop = useIsDesktop();
  const shell = useShell();
  const router = useRouter();
  const queue = useCaptureQueue();
  const notes = useNotes(filtersFor(shell.view, shell.notebookId, shell.sort));
  const rows = notes.data ?? [];

  if (desktop) {
    return (
      <View className="flex-1 items-center justify-center bg-bg">
        <EmptyState
          title={strings.list.pickANote}
          body={strings.list.pickANoteBody}
          icon="file-text"
          action={{ label: strings.list.newNote, onPress: () => shell.newNote() }}
        />
      </View>
    );
  }

  return (
    <Screen>
      <MobileListHeader title={strings.list.title} count={rows.length} archiveLink />
      <NoteListBody
        notes={rows}
        loading={notes.isPending}
        density={shell.density}
        queued={queue.queued}
        onOpen={(id) => router.push(`/(app)/note/${id}`)}
        emptyTitle={strings.list.emptyTitle}
        emptyBody={strings.list.emptyBody}
        onNewNote={shell.openCapture}
      />
      {queue.queued.length > 0 ? (
        <Text className="px-[20px] py-[6px] font-sans text-[11px] text-neutral-600">
          {strings.capture.queued(queue.queued.length)}
        </Text>
      ) : null}
      <CaptureFab />
      <CaptureSheet
        visible={shell.captureVisible}
        onClose={shell.closeCapture}
        notebooks={shell.notebooks}
        defaultNotebookId={shell.notebookId}
      />
    </Screen>
  );
}
