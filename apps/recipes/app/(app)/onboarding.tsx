import { useCreateFamily } from "@fm/api";
import { useAuth } from "@fm/auth";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, View } from "react-native";

import { useI18n } from "../../components/i18n/index.tsx";
import { Display, Field, PrimaryButton, Screen } from "../../components/organic/ui.tsx";

export default function OnboardingScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const { refreshNow } = useAuth();
  const createFamily = useCreateFamily();
  const [name, setName] = useState("");

  return (
    <Screen>
      <View className="flex-1 justify-center gap-[20px] px-[24px]">
        <View>
          <Display size={33}>{t("onboarding.title")}</Display>
          <Text className="mt-[10px] font-fig text-[15.5px] leading-[23px] text-neutral-700">
            {t("onboarding.body")}
          </Text>
        </View>
        <Field
          label={t("onboarding.familyName")}
          value={name}
          onChangeText={setName}
          placeholder={t("onboarding.familyNamePlaceholder")}
        />
        <PrimaryButton
          title={createFamily.isPending ? t("onboarding.creating") : t("onboarding.create")}
          disabled={name.trim() === "" || createFamily.isPending}
          onPress={() =>
            createFamily.mutate(name.trim(), {
              // The access token's family_id claim is baked in at issuance; refresh it before
              // navigating back so the family-scoped screens don't hit the same 403 again.
              onSuccess: () => void refreshNow().then(() => router.replace("/(app)")),
            })
          }
        />
      </View>
    </Screen>
  );
}
