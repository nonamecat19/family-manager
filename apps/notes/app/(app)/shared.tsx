import { useSharedWithMe } from "@fm/api";
import { useRouter } from "expo-router";
import { Pressable, ScrollView, Text, View } from "react-native";

import { relative, strings } from "../../components/i18n/index.ts";
import { EmptyState, Icon, Screen, organic } from "@fm/ui";
import { NoteListBody, useIsDesktop, useShell } from "./_layout.tsx";

export default function SharedScreen() {
  const router = useRouter();
  const shell = useShell();
  const desktop = useIsDesktop();
  const shared = useSharedWithMe();

  const notes = shared.data?.notes ?? [];
  const notebooks = shared.data?.notebooks ?? [];

  if (desktop) {
    return (
      <View className="flex-1 bg-bg px-10 py-9">
        <Text className="pb-lg font-cap text-22 text-fg">{strings.rail.sharedWithMe}</Text>
        {notebooks.length === 0 && notes.length === 0 ? (
          <EmptyState title={strings.list.emptySharedTitle} body={strings.list.emptySharedBody} />
        ) : null}
        <ScrollView>
          {notebooks.map((notebook) => (
            <Pressable
              key={notebook.id}
              accessibilityRole="button"
              accessibilityLabel={notebook.name}
              onPress={() => {
                shell.select("notebook", notebook.id);
                router.push(`/(app)/notebook/${notebook.id}`);
              }}
              className="flex-row items-center gap-2.5 py-2.5"
            >
              <Icon name="folder-simple" size={16} color={organic.accent[600]} />
              <Text className="font-fig text-14 text-fg">{notebook.name}</Text>
              <Text className="ml-auto font-fig text-11.5 text-neutral-600">
                {strings.list.noteCount(notebook.noteCount)}
              </Text>
            </Pressable>
          ))}
          {notes.map((note) => (
            <Pressable
              key={note.id}
              accessibilityRole="button"
              accessibilityLabel={note.title || strings.common.untitled}
              onPress={() => router.push(`/(app)/note/${note.id}`)}
              className="gap-xs py-2.5"
            >
              <Text className="font-fig-med text-14 text-fg">
                {note.title || strings.common.untitled}
              </Text>
              <Text className="font-fig text-12 text-neutral-700">
                {relative(note.updatedAt)}
              </Text>
            </Pressable>
          ))}
        </ScrollView>
      </View>
    );
  }

  return (
    <Screen>
      <View className="flex-row items-end gap-sm px-5 pb-2.5 pt-1.5">
        <Text className="font-cap text-26 text-fg" style={{ letterSpacing: -0.5 }}>
          {strings.rail.sharedWithMe}
        </Text>
        <Text className="pb-xs font-fig text-12 text-neutral-600">{notes.length}</Text>
      </View>
      {notebooks.length > 0 ? (
        <View className="px-5 pb-sm">
          {notebooks.map((notebook) => (
            <Pressable
              key={notebook.id}
              accessibilityRole="button"
              accessibilityLabel={notebook.name}
              onPress={() => {
                shell.select("notebook", notebook.id);
                router.push(`/(app)/notebook/${notebook.id}`);
              }}
              className="flex-row items-center gap-2.5 py-sm"
            >
              <Icon name="folder-simple" size={15} color={organic.accent[600]} />
              <Text className="font-fig text-14 text-fg">{notebook.name}</Text>
              <Text className="ml-auto font-fig text-11.5 text-neutral-600">
                {strings.list.noteCount(notebook.noteCount)}
              </Text>
            </Pressable>
          ))}
        </View>
      ) : null}
      <NoteListBody
        notes={notes}
        loading={shared.isPending}
        density={shell.density}
        onOpen={(id) => router.push(`/(app)/note/${id}`)}
        emptyTitle={strings.list.emptySharedTitle}
        emptyBody={strings.list.emptySharedBody}
        onNewNote={shell.openCapture}
      />
    </Screen>
  );
}
