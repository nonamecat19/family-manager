import { useFamily } from "@fm/api";
import { Code } from "@connectrpc/connect";
import { Slot, Tabs, useRouter, useSegments } from "expo-router";
import { Text, View } from "react-native";

import { BasketProvider } from "../../components/basket.tsx";
import { Icon, type IconName } from "../../components/organic/icons.tsx";
import { organic } from "../../components/organic/tokens.ts";
import { Display, PrimaryButton, Screen } from "../../components/organic/ui.tsx";

export default function AppLayout() {
  const family = useFamily();
  const router = useRouter();
  const segments = useSegments();
  const isOnboarding = segments[segments.length - 1] === "onboarding";

  if (family.isPending) return <Kitchen label="Setting the table…" />;

  if (family.isError) {
    const code = (family.error as { code?: Code }).code;
    if (code === Code.FailedPrecondition) {
      // The onboarding screen lives under this same gate, so without this branch pushing to
      // it just changes the URL — this component still short-circuits before any outlet renders.
      if (isOnboarding) return <Slot />;
      return (
        <Screen>
          <View className="flex-1 justify-center gap-[18px] px-[22px]">
            <Display size={30}>No household{"\n"}yet</Display>
            <Text className="font-fig text-[15.5px] leading-[23px] text-neutral-700">
              Create one to start cooking, or accept an invitation from a family member.
            </Text>
            <PrimaryButton
              title="Create a household"
              onPress={() => router.push("/(app)/onboarding")}
            />
          </View>
        </Screen>
      );
    }
    return (
      <Screen>
        <View className="flex-1 justify-center gap-[18px] px-[22px]">
          <Display size={30}>Something{"\n"}went wrong</Display>
          <Text className="font-fig text-[15.5px] leading-[23px] text-neutral-700">
            {family.error.message}
          </Text>
          <PrimaryButton title="Try again" onPress={() => void family.refetch()} />
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
            fontFamily: "Figtree_800ExtraBold",
            fontSize: 11,
            letterSpacing: 0.2,
          },
        }}
      >
        <Tabs.Screen name="index" options={{ title: "Home", tabBarIcon: tabIcon("home") }} />
        <Tabs.Screen name="recipes" options={{ title: "Recipes", tabBarIcon: tabIcon("book") }} />
        <Tabs.Screen name="meal-plan" options={{ title: "Plan", tabBarIcon: tabIcon("plan") }} />
        <Tabs.Screen name="basket" options={{ title: "List", tabBarIcon: tabIcon("cart") }} />
        <Tabs.Screen name="settings" options={{ title: "You", tabBarIcon: tabIcon("user") }} />
        {/* Reachable from Home and the profile, but not a tab of its own — the design gives
            favourites a card, not a fifth of the bar. */}
        <Tabs.Screen name="favorites" options={{ href: null }} />
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
  return function TabIcon({ color }: { color: string }) {
    return <Icon name={name} size={25} color={color} width={2.4} />;
  };
}

/** The gate's own loading state, so the app never flashes a blue-grey spinner screen. */
function Kitchen({ label }: { label: string }) {
  return (
    <Screen>
      <View className="flex-1 items-center justify-center gap-[10px]">
        <Display size={26}>Family Recipes</Display>
        <Text className="font-fig-semi text-[14px] text-neutral-600">{label}</Text>
      </View>
    </Screen>
  );
}
