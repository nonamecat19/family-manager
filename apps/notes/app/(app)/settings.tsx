import { toDisplayError, useFamily, useNotificationPreferences, useSetNotificationPreferences, useTelegramLink } from "@fm/api";
import { useAuth } from "@fm/auth";
import {
  Avatar,
  DangerLink,
  NotificationsSection,
  PrimaryButton,
  Screen,
  ScreenHeader,
  ScrollBody,
  SettingsGroup,
  SettingsLinkRow,
  SettingsSection,
  TelegramSection,
} from "@fm/ui";
import Constants from "expo-constants";
import * as Linking from "expo-linking";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { offlineCopy, useCaptureQueue } from "../../components/offline/index.ts";

const SIGN_OUT_FLUSH_MS = 4_000;

const telegramBot = (Constants.expoConfig?.extra as { telegramBot?: string } | undefined)?.telegramBot;

export default function SettingsScreen() {
  const family = useFamily();
  const router = useRouter();
  const { signOut } = useAuth();
  const queue = useCaptureQueue();
  const [signingOut, setSigningOut] = useState(false);
  const telegram = useTelegramLink({ bot: telegramBot, open: Linking.openURL });
  const notifications = useNotificationPreferences();
  const setNotifications = useSetNotificationPreferences();
  const members = family.data?.members ?? [];
  const telegramError = telegram.error
    ? toDisplayError(telegram.error, strings.settings.telegramFailed).message
    : null;

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
      <ScrollBody>
        <ScreenHeader title={strings.settings.title} />

        <SettingsSection title={strings.settings.family}>
          <SettingsGroup className="py-3.5">
            <Text className="font-fig-bold text-15.5 text-fg">{family.data?.family?.name ?? ""}</Text>
          </SettingsGroup>
        </SettingsSection>

        <SettingsSection title={strings.settings.members}>
          <SettingsGroup className="py-xs">
            {members.map((member, i) => (
              <View
                key={member.userId}
                className={`flex-row items-center gap-2.5 py-3 ${
                  i === members.length - 1 ? "" : "border-b border-divider"
                }`}
              >
                <Avatar name={member.displayName || member.email} size={32} />
                <View className="flex-1">
                  <Text className="font-fig-bold text-14.5 text-fg" numberOfLines={1}>
                    {member.displayName || member.email}
                  </Text>
                  <Text className="font-fig text-11.5 text-neutral-600">{member.email}</Text>
                </View>
              </View>
            ))}
          </SettingsGroup>
        </SettingsSection>

        <SettingsSection title={strings.settings.offlineQueue}>
          <SettingsGroup className="gap-2.5 py-3.5">
            <Text className="font-fig text-13.5 text-neutral-700">
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
          </SettingsGroup>
        </SettingsSection>

        {queue.rejected.length > 0 ? (
          <SettingsSection title={offlineCopy.refusedTitle}>
            <SettingsGroup className="gap-3 py-3.5">
              <Text className="font-fig text-13.5 text-neutral-700">
                {offlineCopy.refusedBody(queue.rejected.length)}
              </Text>
              {queue.rejected.map((item) => (
                <View key={item.clientId} className="gap-1.5">
                  <Text className="font-fig-bold text-14 text-fg">
                    {item.title || strings.common.untitled}
                  </Text>
                  <Text className="font-fig text-11.5 text-neutral-600">
                    {item.rejection?.reason ?? ""}
                  </Text>
                  <PrimaryButton
                    title={offlineCopy.refile}
                    disabled={queue.syncing || !queue.online}
                    onPress={() => void queue.refile(item.clientId)}
                  />
                </View>
              ))}
            </SettingsGroup>
          </SettingsSection>
        ) : null}

        <TelegramSection
          strings={{
            title: strings.settings.telegram,
            hint: strings.settings.telegramHint,
            waiting: strings.settings.telegramWaiting,
            connected: strings.settings.telegramConnected,
            notConnected: strings.settings.telegramNotConnected,
            connect: strings.settings.telegramConnect,
            disconnect: strings.settings.telegramDisconnect,
          }}
          telegram={telegram}
          error={telegramError}
        />

        <SettingsSection title={strings.settings.account}>
          <View className="gap-2.5">
            <SettingsLinkRow
              label={strings.settings.connectedAccounts}
              onPress={() => router.push("/(app)/connected-accounts")}
            />
            <SettingsLinkRow
              label={strings.settings.approveDevice}
              onPress={() => router.push("/(app)/approve-device")}
            />
          </View>
        </SettingsSection>

        <NotificationsSection
          title={strings.settings.notifications}
          failedText={strings.settings.notificationsLoadFailed}
          isError={notifications.isError}
          topics={notifications.data?.topics ?? []}
          muted={notifications.data?.muted ?? []}
          onChange={(next) => setNotifications.mutate(next)}
        />

        <SettingsSection title={strings.settings.about}>
          <Text className="font-fig text-13 leading-[20px] text-neutral-700">
            {strings.settings.aboutBody}
          </Text>
        </SettingsSection>

        <View className="gap-sm">
          {unsynced > 0 ? (
            <Text className="text-center font-fig text-12 text-neutral-700">
              {offlineCopy.unsyncedOnSignOut(unsynced)}
            </Text>
          ) : null}
          <DangerLink
            title={signingOut ? offlineCopy.signingOut : strings.settings.signOut}
            onPress={() => {
              if (signingOut) return;
              void endSession();
            }}
          />
        </View>
      </ScrollBody>
    </Screen>
  );
}
