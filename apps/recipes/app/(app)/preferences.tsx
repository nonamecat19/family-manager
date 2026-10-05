import { toDisplayError, useNotificationPreferences, useSetNotificationPreferences, useTelegramLink } from "@fm/api";
import Constants from "expo-constants";
import * as Linking from "expo-linking";
import { useRouter } from "expo-router";
import {
  LanguageSection,
  LinkSection,
  NotificationsSection,
  ScreenHeader,
  ScrollBody,
  Screen,
  TelegramSection,
} from "@fm/ui";

import { useI18n, type Locale } from "../../components/i18n/index.tsx";

const telegramBot = (Constants.expoConfig?.extra as { telegramBot?: string } | undefined)?.telegramBot;

export default function PreferencesScreen() {
  const router = useRouter();
  const { t, locale, setLocale } = useI18n();
  const telegram = useTelegramLink({ bot: telegramBot, open: Linking.openURL });
  const telegramError = telegram.error
    ? toDisplayError(telegram.error, t("preferences.telegramFailed")).message
    : null;
  const notificationPrefs = useNotificationPreferences();
  const setNotificationPrefs = useSetNotificationPreferences();

  const options: { value: Locale; label: string }[] = [
    { value: "en", label: t("preferences.english") },
    { value: "uk", label: t("preferences.ukrainian") },
  ];

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader title={t("preferences.title")} onBack={() => router.back()} backLabel={t("common.back")} />

        <LanguageSection
          title={t("preferences.language")}
          hint={t("preferences.languageHint")}
          options={options}
          value={locale}
          onChange={setLocale}
        />

        <TelegramSection
          strings={{
            title: t("preferences.telegram"),
            hint: t("preferences.telegramHint"),
            waiting: t("preferences.telegramWaiting"),
            connected: t("preferences.telegramConnected"),
            notConnected: t("preferences.telegramNotConnected"),
            connect: t("preferences.telegramConnect"),
            disconnect: t("preferences.telegramDisconnect"),
          }}
          telegram={telegram}
          error={telegramError}
        />

        <LinkSection
          title={t("preferences.connectedAccounts")}
          label={t("preferences.connectedAccounts")}
          onPress={() => router.push("/(app)/connected-accounts")}
        />

        <NotificationsSection
          title={t("preferences.notifications")}
          failedText={t("preferences.notificationsFailed")}
          isError={notificationPrefs.isError}
          topics={notificationPrefs.data?.topics ?? []}
          muted={notificationPrefs.data?.muted ?? []}
          onChange={(nextMuted) => setNotificationPrefs.mutate(nextMuted)}
        />

        <LinkSection
          title={t("preferences.approveDevice")}
          label={t("preferences.approveDevice")}
          onPress={() => router.push("/(app)/approve-device")}
        />
      </ScrollBody>
    </Screen>
  );
}
