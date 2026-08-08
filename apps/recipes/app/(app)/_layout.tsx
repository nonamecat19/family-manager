import { useFamily } from "@fm/api";
import { Button, EmptyState, ErrorState, Loading } from "@fm/ui";
import { Tabs, useRouter } from "expo-router";
import { Text } from "react-native";

export default function AppLayout() {
  const family = useFamily();
  const router = useRouter();

  if (family.isPending) return <Loading label="Loading your household…" />;

  if (family.isError) {
    const code = (family.error as { code?: string }).code;
    if (code === "failed_precondition") {
      return (
        <EmptyState
          title="No household yet"
          hint="Create one to start cooking, or accept an invitation from a family member."
          action={
            <Button title="Create a household" onPress={() => router.push("/(app)/onboarding")} />
          }
        />
      );
    }
    return <ErrorState message={family.error.message} onRetry={() => void family.refetch()} />;
  }

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: "#C05621",
      }}
    >
      <Tabs.Screen
        name="index"
        options={{ title: "Recipes", tabBarIcon: () => <Text>🍲</Text> }}
      />
      <Tabs.Screen
        name="favorites"
        options={{ title: "Favorites", tabBarIcon: () => <Text>♥</Text> }}
      />
      <Tabs.Screen
        name="meal-plan"
        options={{ title: "Meal Plan", tabBarIcon: () => <Text>📅</Text> }}
      />
      <Tabs.Screen
        name="settings"
        options={{ title: "Settings", tabBarIcon: () => <Text>⚙</Text> }}
      />
      <Tabs.Screen name="onboarding" options={{ href: null }} />
      <Tabs.Screen name="recipe" options={{ href: null }} />
      <Tabs.Screen name="recipe-edit" options={{ href: null }} />
    </Tabs>
  );
}