import {
  toDisplayError,
  useAccounts,
  useFamily,
  useFinanceMembers,
  useFinanceSettings,
  useNotificationPreferences,
  useSetNotificationPreferences,
  useTelegramLink,
  useTemplates,
  useUpdateFinanceSettings,
  useWidgets,
} from "@fm/api";
import { useAuth } from "@fm/auth";
import Constants from "expo-constants";
import * as Linking from "expo-linking";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { clockTime, currentMember } from "@/components/screens/settings/currentMember.ts";
import { AdvancedSheet, DataSheet, PinSheet, PrivacySheet } from "@/components/screens/settings/sheets.tsx";
import { Avatar, DangerLink, Display, GateMessage, initialOf, LanguageSection, LinkSection, NotificationsSection, PillButton, Screen, ScreenHeader, ScrollBody, StatTile, TelegramSection, SettingsLinkRow } from "@fm/ui";

import type { Locale } from "@/components/i18n";

type SheetName = "privacy" | "pin" | "data" | "advanced";

const telegramBot = (Constants.expoConfig?.extra as { telegramBot?: string } | undefined)?.telegramBot;

export default function SettingsScreen() {
  const { t, locale, setLocale } = useI18n();
  const router = useRouter();
  const { signOut } = useAuth();

  const members = useFinanceMembers();
  const templates = useTemplates();
  const widgets = useWidgets();
  const accounts = useAccounts();
  const settings = useFinanceSettings();
  const family = useFamily();
  const updateSettings = useUpdateFinanceSettings();
  const telegram = useTelegramLink({ bot: telegramBot, open: Linking.openURL });
  const prefs = useNotificationPreferences();
  const setPrefs = useSetNotificationPreferences();

  const [sheet, setSheet] = useState<SheetName | null>(null);

  const queries = [members, templates, widgets, accounts, settings, family];
  const pending = queries.some((query) => query.isPending);
  const failed = queries.find((query) => query.isError);

  if (pending) {
    return (
      <Screen>
        <GateMessage title={t("settings.title")} body={t("common.loadingEllipsis")} />
      </Screen>
    );
  }

  if (failed) {
    const shown = toDisplayError(failed.error, t("common.loadFailed"));
    return (
      <Screen>
        <GateMessage
          title={t("gate.errorTitle")}
          body={shown.message}
          reference={shown.reference ? t("common.errorReference", { ref: shown.reference }) : undefined}
          actionTitle={t("common.tryAgain")}
          onAction={() => queries.forEach((query) => void query.refetch())}
        />
      </Screen>
    );
  }

  const memberList = members.data ?? [];
  const privateOwn = accounts.data?.privateOwn ?? [];
  const me = currentMember(memberList, templates.data, privateOwn);
  const currencyCode = settings.data?.baseCurrencyCode ?? "UAH";
  const version = t("settings.version", { version: Constants.expoConfig?.version ?? "" });
  const familyName = family.data?.family?.name ?? "";
  const templateCount = templates.data?.length ?? 0;
  const widgetCount = widgets.data?.length ?? 0;
  const accountCount =
    (accounts.data?.shared.length ?? 0) + (accounts.data?.privateOwn.length ?? 0);

  const languageOptions: { value: Locale; label: string }[] = [
    { value: "uk", label: t("settings.ukrainian") },
    { value: "en", label: t("settings.english") },
  ];

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader title={t("settings.title")} />

        <View className="flex-row items-center gap-[16px]">
          <Avatar initial={initialOf(me?.displayName || familyName)} size={64} />
          <View className="flex-1">
            <Display size={19}>{me?.displayName || familyName}</Display>
            <Text className="mt-[2px] text-[13px] font-fig-bold text-neutral-600">
              {familyName}
            </Text>
          </View>
        </View>

        <View className="flex-row gap-[10px]">
          <StatTile value={String(templateCount)} label={t("settings.statTemplates")} />
          <StatTile value={String(widgetCount)} label={t("settings.statWidgets")} />
          <StatTile value={String(accountCount)} label={t("settings.statAccounts")} />
        </View>

        <View className="gap-[10px]">
          <SettingsLinkRow
            icon="users-three"
            iconTone="accent"
            label={t("settings.family")}
            meta={t("settings.familyMeta", {
              members: t("common.memberCount", { count: memberList.length }),
            })}
            onPress={() => router.push("/(app)/household")}
          />
          <SettingsLinkRow
            icon="lightning"
            iconTone="accent"
            label={t("settings.templates")}
            meta={t("settings.templatesMeta", {
              count: templateCount,
              name: me?.displayName ?? "",
            })}
            onPress={() => router.push("/(app)/templates")}
          />
          <SettingsLinkRow
            icon="squares-four"
            iconTone="accent"
            label={t("settings.widgets")}
            meta={t("common.activeCount", { count: widgetCount })}
            onPress={() => router.push("/(app)/widgets")}
          />
          <SettingsLinkRow
            icon="chart-bar"
            iconTone="accent"
            label={t("nav.charts")}
            onPress={() => router.push("/(app)/charts")}
          />
          <SettingsLinkRow
            icon="squares-four"
            iconTone="accent"
            label={t("nav.categories")}
            onPress={() => router.push("/(app)/categories")}
          />
          <SettingsLinkRow
            icon="arrows-clockwise"
            iconTone="accent"
            label={t("nav.recurring")}
            onPress={() => router.push("/(app)/recurring")}
          />
          <SettingsLinkRow
            icon="bell"
            iconTone="accent"
            label={t("nav.reminders")}
            onPress={() => router.push("/(app)/reminders")}
          />
        </View>

        <View className="flex-row gap-[10px]">
          <View className="flex-1">
            <PillButton title={t("settings.privacy")} onPress={() => setSheet("privacy")} />
          </View>
          <View className="flex-1">
            <PillButton title={t("settings.pin")} onPress={() => setSheet("pin")} />
          </View>
        </View>

        <LanguageSection
          title={t("settings.language")}
          hint={t("settings.languageHint")}
          options={languageOptions}
          value={locale}
          onChange={setLocale}
        />

        <TelegramSection
          strings={{
            title: t("settings.telegram"),
            hint: t("settings.telegramHint"),
            waiting: t("settings.telegramWaiting"),
            connected: t("settings.telegramConnected"),
            notConnected: t("settings.telegramNotConnected"),
            connect: t("settings.telegramConnect"),
            disconnect: t("settings.telegramDisconnect"),
          }}
          telegram={telegram}
        />

        <NotificationsSection
          title={t("settings.notifications")}
          failedText={t("settings.notificationsFailed")}
          isError={prefs.isError}
          topics={prefs.data?.topics ?? []}
          muted={prefs.data?.muted ?? []}
          onChange={(next) => setPrefs.mutate(next)}
        />

        <LinkSection
          title={t("settings.connectedAccounts")}
          label={t("settings.connectedAccounts")}
          onPress={() => router.push("/(app)/connected-accounts")}
        />

        <LinkSection
          title={t("settings.approveDevice")}
          label={t("settings.approveDevice")}
          onPress={() => router.push("/(app)/approve-device")}
        />

        <SettingsLinkRow
          icon="database"
          label={t("settings.data")}
          onPress={() => setSheet("data")}
        />
        <SettingsLinkRow
          icon="sliders-horizontal"
          label={t("settings.advanced")}
          onPress={() => setSheet("advanced")}
        />

        <Text className="text-center text-[10.5px] text-neutral-600">{version}</Text>

        <DangerLink title={t("settings.signOut")} onPress={() => void signOut()} />
      </ScrollBody>

      <PrivacySheet
        visible={sheet === "privacy"}
        onClose={() => setSheet(null)}
        privateOwn={privateOwn}
        currencyCode={currencyCode}
      />
      <PinSheet
        visible={sheet === "pin"}
        onClose={() => setSheet(null)}
        enabled={settings.data?.pinLockEnabled ?? false}
        onChange={(next) => updateSettings.mutate({ pinLockEnabled: next })}
      />
      <DataSheet
        visible={sheet === "data"}
        onClose={() => setSheet(null)}
        settings={settings.data ?? null}
        syncedAt={t("nav.syncedAt", { time: clockTime(settings.dataUpdatedAt) })}
      />
      <AdvancedSheet visible={sheet === "advanced"} onClose={() => setSheet(null)} version={version} />
    </Screen>
  );
}
