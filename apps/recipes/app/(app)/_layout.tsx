import { toDisplayError, useFamily } from "@fm/api";
import { Code } from "@connectrpc/connect";
import { Slot, Tabs, useRouter, useSegments } from "expo-router";
import { Text, View, type ColorValue } from "react-native";

import { BasketProvider } from "../../components/basket.tsx";
import { I18nProvider, useI18n } from "../../components/i18n/index.tsx";
import { Icon, type IconName } from "../../components/organic/icons.tsx";
import { organic } from "../../components/organic/tokens.ts";
import { Display, PrimaryButton, Screen } from "../../components/organic/ui.tsx";

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

  if (family.isPending) return <Kitchen label={t("kitchen.settingTheTable")} />;

  if (family.isError) {
    const code = (family.error as { code?: Code }).code;
    if (code === Code.FailedPrecondition) {
      // The onboarding screen lives under this same gate, so without this branch pushing to
      // it just changes the URL — this component still short-circuits before any outlet renders.
      if (isOnboarding) return <Slot />;
      return (
        <Screen>
          <View className="flex-1 justify-center gap-[18px] px-[22px]">
            <Display size={30}>{t("kitchen.noHouseholdTitle")}</Display>
            <Text className="font-fig text-[15.5px] leading-[23px] text-neutral-700">
              {t("kitchen.noHouseholdBody")}
            </Text>
            <PrimaryButton
              title={t("kitchen.createHousehold")}
              onPress={() => router.push("/(app)/onboarding")}
            />
          </View>
        </Screen>
      );
    }
    const shown = toDisplayError(family.error, t("common.loadFailed"));
    return (
      <Screen>
        <View className="flex-1 justify-center gap-[18px] px-[22px]">
          <Display size={30}>{t("kitchen.errorTitle")}</Display>
          <Text className="font-fig text-[15.5px] leading-[23px] text-neutral-700">
            {shown.message}
          </Text>
          {shown.reference ? (
            <Text className="font-fig text-[13px] leading-[19px] text-neutral-600">
              {t("common.errorReference", { ref: shown.reference })}
            </Text>
          ) : null}
          <PrimaryButton title={t("common.tryAgain")} onPress={() => void family.refetch()} />
        </View>
      </Screen>
    );
  }

  return (
    <BasketProvider>
      <Tabs
        screenOptions={{
          headerShown: false,
          tabBarActiveTintColor: organic.accent[700],
          tabBarInactiveTintColor: organic.neutral[600],
          tabBarStyle: {
            backgroundColor: organic.neutral[100],
            borderTopColor: organic.neutral[300],
            borderTopWidth: 1,
            height: 78,
            paddingTop: 10,
            paddingBottom: 22,
            elevation: 0,
          },
          tabBarLabelStyle: {
            fontFamily: "NunitoSans_800ExtraBold",
            fontSize: 11,
            letterSpacing: 0.2,
          },
        }}
      >
        <Tabs.Screen name="index" options={{ title: t("tabs.home"), tabBarIcon: tabIcon("home") }} />
        <Tabs.Screen name="recipes" options={{ title: t("tabs.recipes"), tabBarIcon: tabIcon("book") }} />
        <Tabs.Screen name="meal-plan" options={{ title: t("tabs.plan"), tabBarIcon: tabIcon("plan") }} />
        <Tabs.Screen name="basket" options={{ title: t("tabs.list"), tabBarIcon: tabIcon("cart") }} />
        <Tabs.Screen name="settings" options={{ title: t("tabs.you"), tabBarIcon: tabIcon("user") }} />
        {/* Reachable from Home and the profile, but not a tab of its own — the design gives
            favourites a card, not a fifth of the bar. */}
        <Tabs.Screen name="favorites" options={{ href: null }} />
        <Tabs.Screen name="preferences" options={{ href: null }} />
        <Tabs.Screen name="search" options={{ href: null }} />
        <Tabs.Screen name="onboarding" options={{ href: null }} />
        <Tabs.Screen name="recipe/[id]" options={{ href: null }} />
        <Tabs.Screen name="recipe-edit/[id]" options={{ href: null }} />
        {/* Cook mode is full-bleed: the tab bar would sit on top of a dark screen you are
            meant to read from across the kitchen. */}
        <Tabs.Screen name="cook/[id]" options={{ href: null, tabBarStyle: { display: "none" } }} />
      </Tabs>
    </BasketProvider>
  );
}

function tabIcon(name: IconName) {
  // React Navigation hands the tint down as ColorValue. The opaque half of that union only
  // turns up for PlatformColor, which this bar never sets, so narrowing back to a string is safe.
  return function TabIcon({ color }: { color: ColorValue }) {
    return <Icon name={name} size={25} color={color as string} width={2.4} />;
  };
}

/** The gate's own loading state, so the app never flashes a blue-grey spinner screen. */
function Kitchen({ label }: { label: string }) {
  const { t } = useI18n();
  return (
    <Screen>
      <View className="flex-1 items-center justify-center gap-[10px]">
        <Display size={26}>{t("kitchen.appName")}</Display>
        <Text className="font-fig-semi text-[14px] text-neutral-600">{label}</Text>
      </View>
    </Screen>
  );
}
