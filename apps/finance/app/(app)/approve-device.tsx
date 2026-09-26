import { toDisplayError, useApproveDeviceLogin, useDenyDeviceLogin } from "@fm/api";
import { Code, ConnectError } from "@connectrpc/connect";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { Button, Field, Screen, ScreenHeader } from "@/components/nocturne";

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
      <ScreenHeader
        title={t("approveDevice.title")}
        leading={{ icon: "arrow-left", label: t("common.back"), onPress: () => router.back() }}
      />

      <View className="gap-n4 px-n5 pt-n3">
        <Text className="text-[13px] text-neutral-400">{t("approveDevice.body")}</Text>

        {decided ? (
          <Text className="text-[14.5px] text-fg">
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
              autoCorrect={false}
              maxLength={9}
            />

            {error ? <Text className="text-[12px] text-overspend">{error}</Text> : null}

            <View className="gap-n3">
              <Button
                title={t("approveDevice.approve")}
                disabled={!canSubmit}
                onPress={() => approve.mutate(code, { onSuccess: () => setDecided("approved") })}
              />
              <Button
                title={t("approveDevice.deny")}
                variant="ghost"
                disabled={!canSubmit}
                onPress={() => deny.mutate(code, { onSuccess: () => setDecided("denied") })}
              />
            </View>
          </>
        )}
      </View>
    </Screen>
  );
}
