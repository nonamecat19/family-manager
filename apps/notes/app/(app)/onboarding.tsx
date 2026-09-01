import { useCreateFamily } from "@fm/api";
import { useAuth } from "@fm/auth";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, TextInput, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { PrimaryButton, Screen, nocturne } from "../../components/nocturne/index.ts";

/** Create-a-family, the same two fields apps/recipes uses, repainted in Nocturne. */
export default function OnboardingScreen() {
  const router = useRouter();
  const { refreshNow } = useAuth();
  const createFamily = useCreateFamily();
  const [name, setName] = useState("");

  return (
    <Screen>
      <View className="flex-1 justify-center gap-[20px] px-[24px]">
        <View className="gap-[10px]">
          <Text className="font-semi text-[30px] text-fg" style={{ letterSpacing: -0.6 }}>
            {strings.onboarding.title}
          </Text>
          <Text className="font-sans text-[15px] leading-[23px] text-neutral-400">
            {strings.onboarding.body}
          </Text>
        </View>

        <View className="gap-[6px]">
          <Text className="font-med text-[12px] text-neutral-500">
            {strings.onboarding.familyName}
          </Text>
          <TextInput
            value={name}
            onChangeText={setName}
            placeholder={strings.onboarding.familyNamePlaceholder}
            placeholderTextColor={nocturne.neutral[600]}
            accessibilityLabel={strings.onboarding.familyName}
            className="h-[46px] rounded-md border border-neutral-800 bg-surface px-[14px] font-sans text-[15px] text-fg"
          />
        </View>

        <PrimaryButton
          title={createFamily.isPending ? strings.onboarding.creating : strings.onboarding.create}
          disabled={name.trim() === "" || createFamily.isPending}
          onPress={() =>
            createFamily.mutate(name.trim(), {
              // The access token's family_id claim is baked in at issuance; refresh it before
              // navigating back, or the family-scoped screens hit the same precondition again.
              onSuccess: () => void refreshNow().then(() => router.replace("/(app)")),
            })
          }
        />
      </View>
    </Screen>
  );
}
