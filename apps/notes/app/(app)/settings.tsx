import { useFamily } from "@fm/api";
import { useAuth } from "@fm/auth";
import { useState } from "react";
import { ScrollView, Text, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { offlineCopy, useCaptureQueue } from "../../components/offline/index.ts";
import { Avatar, Divider, PrimaryButton, Screen } from "../../components/nocturne/index.ts";

const SIGN_OUT_FLUSH_MS = 4_000;

export default function SettingsScreen() {
  const family = useFamily();
  const { signOut } = useAuth();
  const queue = useCaptureQueue();
  const [signingOut, setSigningOut] = useState(false);
  const members = family.data?.members ?? [];

  const endSession = async () => {
    setSigningOut(true);
    try {
      if (queue.online && queue.pending.length > 0) {
        await Promise.race([
          queue.flush(),
          new Promise((resolve) => setTimeout(resolve, SIGN_OUT_FLUSH_MS)),
        ]);
      }
    } catch (error) {
      console.warn("[notes] the last sync before sign-out did not finish", error);
    }
    try {
      await signOut();
    } finally {
      setSigningOut(false);
    }
  };

  const unsynced = queue.queued.length;

  return (
    <Screen>
      <ScrollView contentContainerClassName="gap-[20px] px-[20px] py-[18px]">
        <Text className="font-med text-[26px] text-fg" style={{ letterSpacing: -0.5 }}>
          {strings.settings.title}
        </Text>

        <View className="gap-[8px]">
          <Section label={strings.settings.family} />
          <Text className="font-sans text-[15px] text-fg">{family.data?.family?.name ?? ""}</Text>
        </View>

        <Divider />

        <View className="gap-[10px]">
          <Section label={strings.settings.members} />
          {members.map((member) => (
            <View key={member.userId} className="flex-row items-center gap-[10px]">
              <Avatar name={member.displayName || member.email} size={26} />
              <View>
                <Text className="font-sans text-[14px] text-fg">
                  {member.displayName || member.email}
                </Text>
                <Text className="font-sans text-[11.5px] text-neutral-600">{member.email}</Text>
              </View>
            </View>
          ))}
        </View>

        <Divider />

        <View className="gap-[10px]">
          <Section label={strings.settings.offlineQueue} />
          <Text className="font-sans text-[13px] text-neutral-400">
            {queue.pending.length === 0
              ? strings.settings.queueEmpty
              : strings.capture.queued(queue.pending.length)}
          </Text>
          {queue.pending.length > 0 ? (
            <PrimaryButton
              title={strings.settings.flushNow}
              disabled={queue.syncing || !queue.online}
              onPress={() => void queue.flush()}
            />
          ) : null}
        </View>

        {queue.rejected.length > 0 ? (
          <>
            <Divider />
            <View className="gap-[10px]">
              <Section label={offlineCopy.refusedTitle} />
              {
}
              <Text className="font-sans text-[13px] text-neutral-400">
                {offlineCopy.refusedBody(queue.rejected.length)}
              </Text>
              {queue.rejected.map((item) => (
                <View key={item.clientId} className="gap-[6px]">
                  <Text className="font-sans text-[14px] text-fg">
                    {item.title || strings.common.untitled}
                  </Text>
                  <Text className="font-sans text-[11.5px] text-neutral-600">
                    {item.rejection?.reason ?? ""}
                  </Text>
                  <PrimaryButton
                    title={offlineCopy.refile}
                    disabled={queue.syncing || !queue.online}
                    onPress={() => void queue.refile(item.clientId)}
                  />
                </View>
              ))}
            </View>
          </>
        ) : null}

        <Divider />

        <View className="gap-[10px]">
          <Section label={strings.settings.about} />
          <Text className="font-sans text-[13px] leading-[20px] text-neutral-500">
            {strings.settings.aboutBody}
          </Text>
        </View>

        <View className="gap-[8px]">
          {unsynced > 0 ? (
            <Text className="font-sans text-[12px] text-neutral-500">
              {offlineCopy.unsyncedOnSignOut(unsynced)}
            </Text>
          ) : null}
          <PrimaryButton
            title={signingOut ? offlineCopy.signingOut : strings.settings.signOut}
            disabled={signingOut}
            onPress={() => void endSession()}
          />
        </View>
      </ScrollView>
    </Screen>
  );
}

function Section({ label }: { label: string }) {
  return (
    <Text className="font-semi text-[10px] uppercase text-neutral-500" style={{ letterSpacing: 1 }}>
      {label}
    </Text>
  );
}
