import { toDisplayError, useFamily } from "@fm/api";
import { Code } from "@connectrpc/connect";
import { Tabs, useRouter, useSegments } from "expo-router";
import { GateMessage, Screen, tabIcon, useTabScreenOptions } from "@fm/ui";

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

  if (family.isPending && !isOnboarding) {
    return (
      <Screen>
        <GateMessage title={t("gate.appName")} body={t("gate.preparing")} />
      </Screen>
    );
  }

  if (family.isError && !isOnboarding) {
    const code = (family.error as { code?: Code }).code;
    if (code === Code.FailedPrecondition) {
      return (
        <Screen>
          <GateMessage
            title={t("gate.noHouseholdTitle")}
            body={t("gate.noHouseholdBody")}
            actionTitle={t("gate.createHousehold")}
            onAction={() => router.push("/(app)/onboarding")}
          />
        </Screen>
      );
    }

    const shown = toDisplayError(family.error, t("common.loadFailed"));
    return (
      <Screen>
        <GateMessage
          title={t("gate.errorTitle")}
          body={shown.message}
          reference={shown.reference ? t("common.errorReference", { ref: shown.reference }) : undefined}
          actionTitle={t("common.tryAgain")}
          onAction={() => void family.refetch()}
        />
      </Screen>
    );
  }

  return (
    <Tabs screenOptions={useTabScreenOptions()} backBehavior="history">
      <Tabs.Screen name="index" options={{ title: t("tabs.home"), tabBarIcon: tabIcon("squares-four") }} />
      <Tabs.Screen
        name="transactions"
        options={{ title: t("tabs.transactions"), tabBarIcon: tabIcon("receipt") }}
        listeners={freshOnTabPress("transactions")}
      />
      <Tabs.Screen
        name="add"
        options={{ title: t("tabs.add"), tabBarIcon: tabIcon("plus") }}
        listeners={freshOnTabPress("add")}
      />
      <Tabs.Screen
        name="accounts"
        options={{ title: t("tabs.accounts"), tabBarIcon: tabIcon("wallet") }}
      />
      <Tabs.Screen name="settings" options={{ title: t("tabs.you"), tabBarIcon: tabIcon("user") }} />
      <Tabs.Screen name="charts" options={{ href: null }} />
      <Tabs.Screen name="categories" options={{ href: null }} />
      <Tabs.Screen name="household" options={{ href: null }} />
      <Tabs.Screen name="templates" options={{ href: null }} />
      <Tabs.Screen name="recurring" options={{ href: null }} />
      <Tabs.Screen name="investments" options={{ href: null }} />
      <Tabs.Screen name="installments" options={{ href: null }} />
      <Tabs.Screen name="reminders" options={{ href: null }} />
      <Tabs.Screen name="subscriptions" options={{ href: null }} />
      <Tabs.Screen name="widgets" options={{ href: null }} />
      <Tabs.Screen name="connected-accounts" options={{ href: null }} />
      <Tabs.Screen name="approve-device" options={{ href: null }} />
      <Tabs.Screen name="members/spending" options={{ href: null }} />
      <Tabs.Screen name="onboarding" options={{ href: null, tabBarStyle: { display: "none" } }} />
    </Tabs>
  );
}

function freshOnTabPress(name: string) {
  return ({ navigation }: { navigation: { navigate: (route: string, params: object) => void } }) => ({
    tabPress: (event: { preventDefault: () => void }) => {
      event.preventDefault();
      navigation.navigate(name, {});
    },
  });
}
