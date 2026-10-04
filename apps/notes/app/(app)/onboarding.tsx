import { useCreateFamily } from "@fm/api";
import { useAuth } from "@fm/auth";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { Display, Field, PrimaryButton, Screen } from "@fm/ui";

export default function OnboardingScreen() {
  const router = useRouter();
  const { refreshNow } = useAuth();
  const createFamily = useCreateFamily();
  const [name, setName] = useState("");

  return (
    <Screen>
      <View className="flex-1 justify-center gap-[20px] px-[24px]">
        <View>
          <Display size={33}>{strings.onboarding.title}</Display>
          <Text className="mt-[10px] font-fig text-[15.5px] leading-[23px] text-neutral-700">
            {strings.onboarding.body}
          </Text>
        </View>
        <Field
          label={strings.onboarding.familyName}
          value={name}
          onChangeText={setName}
          placeholder={strings.onboarding.familyNamePlaceholder}
        />
        <PrimaryButton
          title={createFamily.isPending ? strings.onboarding.creating : strings.onboarding.create}
          disabled={name.trim() === "" || createFamily.isPending}
          onPress={() =>
            createFamily.mutate(name.trim(), {
              onSuccess: () => void refreshNow().then(() => router.replace("/(app)")),
            })
          }
        />
      </View>
    </Screen>
  );
}
