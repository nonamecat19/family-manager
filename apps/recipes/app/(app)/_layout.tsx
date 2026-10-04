import { toDisplayError, useFamily } from "@fm/api";
import { Code } from "@connectrpc/connect";
import { Slot, Tabs, useRouter, useSegments } from "expo-router";
import { BootSplash, GateMessage, tabIcon, useTabScreenOptions } from "@fm/ui";

import { BasketProvider } from "../../components/basket.tsx";
import { I18nProvider, useI18n } from "../../components/i18n/index.tsx";

export default function AppLayout() {
  return (
    <I18nProvider>
      <Gate />
    </I18nProvider>
  );
}

function Gate() {
  const { t } = useI18n();
  const family = useFamily();
  const router = useRouter();
  const segments = useSegments();
  const isOnboarding = segments[segments.length - 1] === "onboarding";
  const tabScreenOptions = useTabScreenOptions();

  if (family.isPending) return <BootSplash title={t("kitchen.appName")} label={t("kitchen.settingTheTable")} />;

  if (family.isError) {
    const code = (family.error as { code?: Code }).code;
    if (code === Code.FailedPrecondition) {
      if (isOnboarding) return <Slot />;
      return (
        <GateMessage
          title={t("kitchen.noHouseholdTitle")}
          body={t("kitchen.noHouseholdBody")}
          actionTitle={t("kitchen.createHousehold")}
          onAction={() => router.push("/(app)/onboarding")}
        />
      );
    }
    const shown = toDisplayError(family.error, t("common.loadFailed"));
    return (
      <GateMessage
        title={t("kitchen.errorTitle")}
        body={shown.message}
        reference={shown.reference ? t("common.errorReference", { ref: shown.reference }) : undefined}
        actionTitle={t("common.tryAgain")}
        onAction={() => void family.refetch()}
      />
    );
  }

  return (
    <BasketProvider>
      <Tabs screenOptions={tabScreenOptions}>
        <Tabs.Screen name="index" options={{ title: t("tabs.home"), tabBarIcon: tabIcon("home") }} />
        <Tabs.Screen name="recipes" options={{ title: t("tabs.recipes"), tabBarIcon: tabIcon("book") }} />
        <Tabs.Screen name="meal-plan" options={{ title: t("tabs.plan"), tabBarIcon: tabIcon("plan") }} />
        <Tabs.Screen name="basket" options={{ title: t("tabs.list"), tabBarIcon: tabIcon("cart") }} />
        <Tabs.Screen name="settings" options={{ title: t("tabs.you"), tabBarIcon: tabIcon("user") }} />
        <Tabs.Screen name="favorites" options={{ href: null }} />
        <Tabs.Screen name="preferences" options={{ href: null }} />
        <Tabs.Screen name="connected-accounts" options={{ href: null }} />
        <Tabs.Screen name="approve-device" options={{ href: null }} />
        <Tabs.Screen name="search" options={{ href: null }} />
        <Tabs.Screen name="onboarding" options={{ href: null }} />
        <Tabs.Screen name="recipe/[id]" options={{ href: null }} />
        <Tabs.Screen name="recipe-edit/[id]" options={{ href: null }} />
        <Tabs.Screen name="cook/[id]" options={{ href: null, tabBarStyle: { display: "none" } }} />
      </Tabs>
    </BasketProvider>
  );
}
