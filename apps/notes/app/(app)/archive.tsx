import { useNotes } from "@fm/api";
import { useRouter } from "expo-router";
import { useEffect } from "react";
import { View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { EmptyState, IconButton, Screen, nocturne } from "../../components/nocturne/index.ts";
import {
  MobileListHeader,
  NoteListBody,
  filtersFor,
  useIsDesktop,
  useShell,
} from "./_layout.tsx";

/**
 * The archive, as a screen.
 *
 * It exists because archiving is offered on BOTH layouts — the note's overflow menu has it
 * wherever the note is open — while the only way back in used to be the desktop rail's Archive
 * row. On a phone that made archiving a one-way trip: the note left every list and no route
 * showed it again, so "take it out of the archive" was a menu item on a screen the user could
 * no longer reach. This is that route.
 *
 * The list is ListNotes with archived_only, the same filter the rail row uses — one
 * `filtersFor` call, so the two layouts cannot drift into showing different archives.
 *
 * On MOBILE this screen deliberately does NOT call `shell.select("archive")`. The shell's view
 * is what the notes TAB renders, and moving it here would leave the tab showing archived notes
 * under the heading "All notes" once the user went back. A pushed screen owns its own query.
 * On DESKTOP the opposite is true: the rail and the list pane are the chrome around this
 * route, so a deep link to /archive has to light the rail row up, exactly as notebook/[id]
 * does — otherwise the URL and the highlighted row disagree.
 */
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
          icon="archive"
        />
      </View>
    );
  }

  return (
    <Screen>
      <View className="flex-row items-center px-[16px] pt-[6px]">
        <IconButton
          icon="caret-left"
          label={strings.note.back}
          size={20}
          color={nocturne.accent.DEFAULT}
          onPress={() => router.back()}
        />
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
