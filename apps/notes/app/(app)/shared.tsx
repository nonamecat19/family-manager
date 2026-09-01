import { useSharedWithMe } from "@fm/api";
import { useRouter } from "expo-router";
import { Pressable, ScrollView, Text, View } from "react-native";

import { relative, strings } from "../../components/i18n/index.ts";
import { EmptyState, Icon, Screen, nocturne } from "../../components/nocturne/index.ts";
import { NoteListBody, useIsDesktop, useShell } from "./_layout.tsx";

/**
 * "Shared with me": everything the caller can see that they do not own. ListSharedWithMe
 * returns two lists — notes and notebooks — and both are drawn, because a shared notebook is
 * how most of the shared notes arrive.
 */
export default function SharedScreen() {
  const router = useRouter();
  const shell = useShell();
  const desktop = useIsDesktop();
  const shared = useSharedWithMe();

  const notes = shared.data?.notes ?? [];
  const notebooks = shared.data?.notebooks ?? [];

  if (desktop) {
    return (
      <View className="flex-1 bg-bg px-[40px] py-[36px]">
        <Text className="pb-[16px] font-med text-[22px] text-fg">{strings.rail.sharedWithMe}</Text>
        {notebooks.length === 0 && notes.length === 0 ? (
          <EmptyState title={strings.list.emptySharedTitle} body={strings.list.emptySharedBody} icon="users-three" />
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
              className="flex-row items-center gap-[10px] py-[10px]"
            >
              <Icon name="folder-simple" size={16} color={nocturne.accent[400]} />
              <Text className="font-sans text-[14px] text-fg">{notebook.name}</Text>
              <Text className="ml-auto font-sans text-[11.5px] text-neutral-600">
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
              className="gap-[4px] py-[10px]"
            >
              <Text className="font-med text-[14px] text-fg">
                {note.title || strings.common.untitled}
              </Text>
              <Text className="font-sans text-[12px] text-neutral-500">
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
      <View className="flex-row items-end gap-[8px] px-[20px] pb-[10px] pt-[6px]">
        <Text className="font-med text-[26px] text-fg" style={{ letterSpacing: -0.5 }}>
          {strings.rail.sharedWithMe}
        </Text>
        <Text className="pb-[4px] font-sans text-[12px] text-neutral-600">{notes.length}</Text>
      </View>
      {notebooks.length > 0 ? (
        <View className="px-[20px] pb-[8px]">
          {notebooks.map((notebook) => (
            <Pressable
              key={notebook.id}
              accessibilityRole="button"
              accessibilityLabel={notebook.name}
              onPress={() => {
                shell.select("notebook", notebook.id);
                router.push(`/(app)/notebook/${notebook.id}`);
              }}
              className="flex-row items-center gap-[10px] py-[8px]"
            >
              <Icon name="folder-simple" size={15} color={nocturne.accent[400]} />
              <Text className="font-sans text-[14px] text-fg">{notebook.name}</Text>
              <Text className="ml-auto font-sans text-[11.5px] text-neutral-600">
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
