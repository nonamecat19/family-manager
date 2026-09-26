import { toDisplayError, useApproveDeviceLogin, useDenyDeviceLogin } from "@fm/api";
import { Code, ConnectError } from "@connectrpc/connect";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, Text, View } from "react-native";

import { useI18n } from "../../components/i18n/index.tsx";
import { organic } from "../../components/organic/tokens.ts";
import { Display, Field, OutlineButton, PrimaryButton, RoundButton, Screen } from "../../components/organic/ui.tsx";

function normalizeUserCode(raw: string): string {
  const letters = raw.toUpperCase().replace(/[^A-Z]/g, "").slice(0, 8);
  return letters.length > 4 ? `${letters.slice(0, 4)}-${letters.slice(4)}` : letters;
}

export default function ApproveDeviceScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const [code, setCode] = useState("");
  const [decided, setDecided] = useState<"approved" | "denied" | null>(null);
  const approve = useApproveDeviceLogin();
  const deny = useDenyDeviceLogin();

  const approveDeviceError = (error: unknown): string => {
    const errCode = error instanceof ConnectError ? error.code : undefined;
    if (errCode === Code.NotFound) return t("approveDevice.notFound");
    if (errCode === Code.ResourceExhausted) return t("approveDevice.tooManyAttempts");
    return toDisplayError(error, t("approveDevice.failed")).message;
  };

  const busy = approve.isPending || deny.isPending;
  const canSubmit = code.length === 9 && !busy;
  const error = approve.isError
    ? approveDeviceError(approve.error)
    : deny.isError
      ? approveDeviceError(deny.error)
      : null;

  return (
    <Screen>
      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerClassName="gap-[20px] px-[22px] pb-[28px] pt-[8px]"
      >
        <View className="flex-row items-center gap-[12px]">
          <RoundButton icon="back" label={t("common.back")} onPress={() => router.back()} />
          <Display size={28}>{t("approveDevice.title")}</Display>
        </View>

        <Text className="font-fig text-[14px] leading-[21px] text-neutral-600">
          {t("approveDevice.body")}
        </Text>

        {decided ? (
          <Text className="font-fig-bold text-[15.5px] text-fg">
            {decided === "approved" ? t("approveDevice.approved") : t("approveDevice.denied")}
          </Text>
        ) : (
          <>
            <Field
              label={t("approveDevice.placeholder")}
              value={code}
              onChangeText={(value) => setCode(normalizeUserCode(value))}
              placeholder={t("approveDevice.placeholder")}
              autoCapitalize="characters"
            />

            {error ? (
              <Text className="font-fig-semi text-[12px]" style={{ color: organic.danger }}>
                {error}
              </Text>
            ) : null}

            <View className="gap-[12px]">
              <PrimaryButton
                title={t("approveDevice.approve")}
                disabled={!canSubmit}
                onPress={() => approve.mutate(code, { onSuccess: () => setDecided("approved") })}
              />
              <OutlineButton
                title={t("approveDevice.deny")}
                onPress={() => {
                  if (!canSubmit) return;
                  deny.mutate(code, { onSuccess: () => setDecided("denied") });
                }}
              />
            </View>
          </>
        )}
      </ScrollView>
    </Screen>
  );
}
