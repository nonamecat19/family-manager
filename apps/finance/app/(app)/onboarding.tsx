import { useAcceptInvitation, useCreateFamily } from "@fm/api";
import { useAuth } from "@fm/auth";
import { Button, Field } from "@fm/ui";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

/** First run: create a household, or join one with an invitation link. */
export default function OnboardingScreen() {
  const router = useRouter();
  const { refreshNow } = useAuth();
  const createFamily = useCreateFamily();
  const acceptInvitation = useAcceptInvitation();

  const [name, setName] = useState("");
  const [token, setToken] = useState("");

  // The access token's family_id claim is baked in at issuance; refresh it before navigating
  // back so the family-scoped screens don't hit the same 403 again.
  const done = () => void refreshNow().then(() => router.replace("/(app)"));

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <ScrollView contentContainerClassName="gap-xl p-xl">
        <View className="gap-xs">
          <Text className="text-display font-bold text-fg dark:text-fg-dark">Set up</Text>
          <Text className="text-body text-muted dark:text-muted-dark">
            A household holds the shared accounts, categories and transactions.
          </Text>
        </View>

        <View className="gap-md">
          <Field
            label="Household name"
            value={name}
            onChangeText={setName}
            placeholder="The Smiths"
          />
          <Button
            title="Create household"
            loading={createFamily.isPending}
            disabled={name.trim() === ""}
            onPress={() => createFamily.mutate(name.trim(), { onSuccess: done })}
          />
          {createFamily.isError ? (
            <Text className="text-caption text-expense">{createFamily.error.message}</Text>
          ) : null}
        </View>

        <View className="h-px bg-border dark:bg-border-dark" />

        <View className="gap-md">
          <Field
            label="Invitation code"
            value={token}
            onChangeText={setToken}
            autoCapitalize="none"
            placeholder="Paste the code you were sent"
          />
          <Button
            title="Join a household"
            variant="secondary"
            loading={acceptInvitation.isPending}
            disabled={token.trim() === ""}
            onPress={() => acceptInvitation.mutate(token.trim(), { onSuccess: done })}
          />
          {acceptInvitation.isError ? (
            <Text className="text-caption text-expense">{acceptInvitation.error.message}</Text>
          ) : null}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
