import {
  fromWire,
  toDisplayError,
  useAccounts,
  useFamily,
  useFinanceMembers,
  useFinanceSettings,
  useTemplates,
  useUpdateFinanceSettings,
  useWidgets,
} from "@fm/api";
import { useAuth } from "@fm/auth";
import Constants from "expo-constants";
import { useRouter, type Href } from "expo-router";
import { useState } from "react";
import { ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  Drawer,
  EmptyState,
  Screen,
  ScreenHeader,
  useDrawerItems,
  type DrawerItem,
} from "@/components/nocturne";
import { SettingsRow } from "@/components/screens/settings/SettingsRow.tsx";
import { clockTime, currentMember } from "@/components/screens/settings/currentMember.ts";
import {
  AdvancedSheet,
  AppearanceSheet,
  DataSheet,
  PinSheet,
  PrivacySheet,
} from "@/components/screens/settings/sheets.tsx";

type SheetName = "privacy" | "pin" | "appearance" | "data" | "advanced";

/**
 * Screen 11 — Settings, with the navigation drawer overlaid on top of it.
 *
 * The drawer is a `Modal` over this screen (the kit's `Drawer`), not an expo-router drawer
 * navigator: the design shows it over a stack, and a navigator would put a second edge
 * gesture on every screen in the app.
 */
export default function SettingsScreen() {
  const { t } = useI18n();
  const router = useRouter();
  const { signOut } = useAuth();

  const members = useFinanceMembers();
  const templates = useTemplates();
  const widgets = useWidgets();
  const accounts = useAccounts();
  const settings = useFinanceSettings();
  const family = useFamily();
  const updateSettings = useUpdateFinanceSettings();

  const drawerItems = useDrawerItems();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [sheet, setSheet] = useState<SheetName | null>(null);

  const queries = [members, templates, widgets, accounts, settings, family];
  const pending = queries.some((query) => query.isPending);
  const failed = queries.find((query) => query.isError);

  const memberList = members.data ?? [];
  const privateOwn = accounts.data?.privateOwn ?? [];
  const me = currentMember(memberList, templates.data, privateOwn);
  const currencyCode = settings.data?.baseCurrencyCode ?? "UAH";
  const version = t("settings.version", { version: Constants.expoConfig?.version ?? "" });

  const openRoute = (href: string) => {
    setDrawerOpen(false);
    router.push(href as Href);
  };

  const onSelectDrawerItem = (item: DrawerItem) => {
    setDrawerOpen(false);
    // The drawer's own entry for this screen just closes it; pushing would stack a second copy.
    if (!item.href || item.id === "settings") return;
    router.push(item.href as Href);
  };

  const header = (
    <ScreenHeader
      title={t("settings.title")}
      gradient
      leading={{ icon: "list", label: t("nav.menu"), onPress: () => setDrawerOpen(true) }}
    />
  );

  const overlay = (
    <Drawer
      visible={drawerOpen}
      onClose={() => setDrawerOpen(false)}
      account={{ name: me?.displayName ?? "", email: me?.email ?? "" }}
      household={{
        name: family.data?.family?.name ?? "",
        // The shared balance only: private accounts are excluded server-side and must never
        // reach a family-scoped total in the UI.
        balance: accounts.data ? fromWire(accounts.data.sharedBalance, currencyCode) : undefined,
      }}
      scopes={[
        { id: "family", label: t("common.family") },
        ...memberList.map((member) => ({ id: member.userId, label: member.displayName })),
      ]}
      // Settings has nothing to scope, so the row navigates instead of pretending to filter:
      // a member opens their spending, "Родина" goes home.
      onSelectScope={(id) => {
        setDrawerOpen(false);
        if (id === "family") {
          router.push("/(app)");
          return;
        }
        router.push({ pathname: "/(app)/members/spending", params: { memberId: id } });
      }}
      items={drawerItems}
      activeId="settings"
      onSelect={onSelectDrawerItem}
      footer={t("nav.syncedAt", { time: clockTime(accounts.dataUpdatedAt) })}
    />
  );

  if (pending) {
    return (
      <Screen>
        {header}
        <View className="flex-1 items-center justify-center">
          <Text className="text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
        </View>
        {overlay}
      </Screen>
    );
  }

  if (failed) {
    const shown = toDisplayError(failed.error, t("common.loadFailed"));
    return (
      <Screen>
        {header}
        <View className="flex-1 justify-center">
          <EmptyState
            icon="gear"
            title={t("gate.errorTitle")}
            body={shown.message}
            action={{
              label: t("common.tryAgain"),
              onPress: () => queries.forEach((query) => void query.refetch()),
            }}
          />
          {shown.reference ? (
            <Text className="px-n6 text-center text-[12px] text-neutral-600">
              {t("common.errorReference", { ref: shown.reference })}
            </Text>
          ) : null}
        </View>
        {overlay}
      </Screen>
    );
  }

  const templateCount = templates.data?.length ?? 0;
  const widgetCount = widgets.data?.length ?? 0;

  return (
    <Screen>
      {header}

      <ScrollView className="flex-1" contentContainerClassName="gap-n3 px-n4 pb-n5 pt-n5">
        <SettingsRow
          icon="users-three"
          tone="accent"
          label={t("settings.family")}
          meta={t("settings.familyMeta", {
            members: t("common.memberCount", { count: memberList.length }),
          })}
          onPress={() => openRoute("/(app)/household")}
        />
        <SettingsRow
          icon="lightning"
          tone="accent"
          label={t("settings.templates")}
          meta={t("settings.templatesMeta", {
            count: templateCount,
            name: me?.displayName ?? "",
          })}
          onPress={() => openRoute("/(app)/templates")}
        />
        <SettingsRow
          icon="squares-four"
          tone="accent"
          label={t("settings.widgets")}
          meta={t("common.activeCount", { count: widgetCount })}
          onPress={() => openRoute("/(app)/widgets")}
        />
        <SettingsRow
          icon="eye-slash"
          tone="accent"
          label={t("settings.privacy")}
          meta={t("common.hiddenAccountCount", { count: privateOwn.length })}
          onPress={() => setSheet("privacy")}
        />

        {/* The canvas breaks the list here: what the household owns, then device preferences. */}
        <View className="h-[8px]" />

        <SettingsRow icon="lock-key" label={t("settings.pin")} onPress={() => setSheet("pin")} />
        <SettingsRow
          icon="palette"
          label={t("settings.appearance")}
          onPress={() => setSheet("appearance")}
        />
        <SettingsRow icon="database" label={t("settings.data")} onPress={() => setSheet("data")} />
        <SettingsRow
          icon="sliders-horizontal"
          label={t("settings.advanced")}
          onPress={() => setSheet("advanced")}
        />
      </ScrollView>

      <Text className="px-n5 pb-n4 text-[10.5px] text-neutral-700">{version}</Text>

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
      <AppearanceSheet visible={sheet === "appearance"} onClose={() => setSheet(null)} />
      <DataSheet
        visible={sheet === "data"}
        onClose={() => setSheet(null)}
        settings={settings.data ?? null}
        syncedAt={t("nav.syncedAt", { time: clockTime(settings.dataUpdatedAt) })}
      />
      <AdvancedSheet
        visible={sheet === "advanced"}
        onClose={() => setSheet(null)}
        version={version}
        onSignOut={() => {
          setSheet(null);
          void signOut();
        }}
      />

      {overlay}
    </Screen>
  );
}
