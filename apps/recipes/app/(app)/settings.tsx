import { useAuth } from "@fm/auth";
import { useRouter } from "expo-router";
import { Pressable, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

export default function SettingsScreen() {
  const { signOut, status } = useAuth();
  const router = useRouter();

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="gap-lg p-lg">
        <Text className="text-display font-bold text-fg dark:text-fg-dark">Settings</Text>

        <View className="gap-md">
          <Text className="text-body text-muted dark:text-muted-dark">
            Family Recipes — your shared cookbook.
          </Text>
        </View>

        <Pressable
          accessibilityRole="button"
          onPress={() => {
            void signOut();
            router.replace("/(auth)/login");
          }}
          className="items-center rounded-lg bg-card dark:bg-card-dark p-md"
        >
          <Text className="text-body text-error">
            {status === "authenticated" ? "Sign out" : "Sign in"}
          </Text>
        </Pressable>
      </View>
    </SafeAreaView>
  );
}