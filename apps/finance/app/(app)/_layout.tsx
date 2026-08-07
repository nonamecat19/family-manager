import { useFamily } from "@fm/api";
import { Button, EmptyState, ErrorState, Loading } from "@fm/ui";
import { Tabs, useRouter } from "expo-router";
import { Text } from "react-native";

/**
 * Everything behind this layout is family-scoped, so the family gate lives here rather than
 * being repeated on each screen: without a household there is no ledger to show.
 */
export default function AppLayout() {
  const family = useFamily();
  const router = useRouter();

  if (family.isPending) return <Loading label="Loading your household…" />;

  if (family.isError) {
    // "Not in a family yet" is a normal state on first launch, not a failure.
    const code = (family.error as { code?: string }).code;
    if (code === "failed_precondition") {
      return (
        <EmptyState
          title="No household yet"
          hint="Create one to start tracking, or accept an invitation from a family member."
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
        tabBarActiveTintColor: "#2F855A",
      }}
    >
      <Tabs.Screen
        name="index"
        options={{ title: "Ledger", tabBarIcon: () => <Text>≡</Text> }}
      />
      <Tabs.Screen
        name="reports"
        options={{ title: "Reports", tabBarIcon: () => <Text>◔</Text> }}
      />
      <Tabs.Screen
        name="accounts"
        options={{ title: "Accounts", tabBarIcon: () => <Text>▣</Text> }}
      />
      <Tabs.Screen
        name="settings"
        options={{ title: "Settings", tabBarIcon: () => <Text>⚙</Text> }}
      />
      <Tabs.Screen name="onboarding" options={{ href: null }} />
      <Tabs.Screen name="transaction" options={{ href: null }} />
    </Tabs>
  );
}
