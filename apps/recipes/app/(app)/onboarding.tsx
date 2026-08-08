import { useCreateFamily } from "@fm/api";
import { Button, Field } from "@fm/ui";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

export default function OnboardingScreen() {
  const router = useRouter();
  const createFamily = useCreateFamily();
  const [name, setName] = useState("");

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="flex-1 justify-center gap-lg p-xl">
        <View className="gap-xs">
          <Text className="text-display font-bold text-fg dark:text-fg-dark">Start your family</Text>
          <Text className="text-body text-muted dark:text-muted-dark">
            Name your household to start collecting recipes together.
          </Text>
        </View>
        <Field label="Family name" value={name} onChangeText={setName} />
        <Button
          title="Create"
          loading={createFamily.isPending}
          disabled={name.trim() === ""}
          onPress={() =>
            createFamily.mutate(name.trim(), {
              onSuccess: () => router.replace("/(app)"),
            })
          }
        />
      </View>
    </SafeAreaView>
  );
}