import { toDisplayError, useFamily } from "@fm/api";
import { Code } from "@connectrpc/connect";
import { Stack, useRouter, useSegments } from "expo-router";
import { Text, View } from "react-native";

import { I18nProvider, useI18n } from "../../components/i18n/index.tsx";
import { Button, EmptyState, nocturne, Screen } from "../../components/nocturne/index.ts";

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

  if (family.isPending) return <Boot label={t("gate.preparing")} />;

  if (family.isError) {
    const code = (family.error as { code?: Code }).code;
    if (code === Code.FailedPrecondition) {
      if (isOnboarding) return <Routes />;
      return (
        <Screen>
          <View className="flex-1 justify-center">
            <EmptyState
              icon="users-three"
              title={t("gate.noHouseholdTitle")}
              body={t("gate.noHouseholdBody")}
              action={{
                label: t("gate.createHousehold"),
                onPress: () => router.push("/(app)/onboarding"),
              }}
            />
          </View>
        </Screen>
      );
    }

    const shown = toDisplayError(family.error, t("common.loadFailed"));
    return (
      <Screen>
        <View className="flex-1 justify-center gap-n4 px-n6">
          <Text className="text-[24px] font-medium leading-[28px] text-fg">{t("gate.errorTitle")}</Text>
          <Text className="text-[13.5px] leading-[21px] text-neutral-500">{shown.message}</Text>
          {shown.reference ? (
            <Text className="text-[12px] text-neutral-600">
              {t("common.errorReference", { ref: shown.reference })}
            </Text>
          ) : null}
          <Button title={t("common.tryAgain")} onPress={() => void family.refetch()} />
        </View>
      </Screen>
    );
  }

  return <Routes />;
}

function Routes() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        contentStyle: { backgroundColor: nocturne.bg },
        animation: "slide_from_right",
      }}
    />
  );
}

function Boot({ label }: { label: string }) {
  const { t } = useI18n();
  return (
    <Screen>
      <View className="flex-1 items-center justify-center gap-n3">
        <Text className="text-[20px] font-medium text-fg">{t("gate.appName")}</Text>
        <Text className="text-[13px] text-neutral-500">{label}</Text>
      </View>
    </Screen>
  );
}
