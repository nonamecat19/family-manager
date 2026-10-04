import { toDisplayError, useApproveDeviceLogin, useDenyDeviceLogin } from "@fm/api";
import { useTheme } from "@fm/ui";
import { Code, ConnectError } from "@connectrpc/connect";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, TextInput, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { IconButton, PrimaryButton, Screen, nocturne } from "../../components/nocturne/index.ts";

function normalizeUserCode(raw: string): string {
  const letters = raw.toUpperCase().replace(/[^A-Z]/g, "").slice(0, 8);
  return letters.length > 4 ? `${letters.slice(0, 4)}-${letters.slice(4)}` : letters;
}

function approveDeviceError(error: unknown): string {
  const code = error instanceof ConnectError ? error.code : undefined;
  if (code === Code.NotFound) return strings.approveDevice.notFound;
  if (code === Code.ResourceExhausted) return strings.approveDevice.tooManyAttempts;
  return toDisplayError(error, strings.approveDevice.failed).message;
}

export default function ApproveDeviceScreen() {
  const router = useRouter();
  const theme = useTheme();
  const [code, setCode] = useState("");
  const [decided, setDecided] = useState<"approved" | "denied" | null>(null);
  const approve = useApproveDeviceLogin();
  const deny = useDenyDeviceLogin();

  const busy = approve.isPending || deny.isPending;
  const canSubmit = code.length === 9 && !busy;
  const error = approve.isError
    ? approveDeviceError(approve.error)
    : deny.isError
      ? approveDeviceError(deny.error)
      : null;

  return (
    <Screen>
      <View className="flex-row items-center gap-[10px] px-[16px] pb-[10px] pt-[6px]">
        <IconButton
          icon="caret-left"
          label={strings.approveDevice.back}
          size={20}
          color={nocturne.accent.DEFAULT}
          onPress={() => router.back()}
        />
        <Text className="font-med text-[17px] text-fg">{strings.approveDevice.title}</Text>
      </View>

      <View className="flex-1 gap-[18px] px-[24px] pt-[12px]">
        <Text className="font-sans text-[14px] leading-[21px] text-neutral-400">
          {strings.approveDevice.body}
        </Text>

        {decided ? (
          <Text className="font-sans text-[15px] text-fg">
            {decided === "approved" ? strings.approveDevice.approved : strings.approveDevice.denied}
          </Text>
        ) : (
          <>
            <TextInput
              value={code}
              onChangeText={(value) => setCode(normalizeUserCode(value))}
              placeholder={strings.approveDevice.placeholder}
              placeholderTextColor={nocturne.neutral[600]}
              autoCapitalize="characters"
              autoCorrect={false}
              accessibilityLabel={strings.approveDevice.placeholder}
              maxLength={9}
              className="h-[46px] rounded-md border border-neutral-800 bg-surface px-[14px] text-center font-sans text-[18px] text-fg"
              style={{ letterSpacing: 3 }}
            />

            {error ? (
              <Text className="font-sans text-[12px]" style={{ color: theme.danger }}>
                {error}
              </Text>
            ) : null}

            <View className="gap-[10px]">
              <PrimaryButton
                title={strings.approveDevice.approve}
                disabled={!canSubmit}
                onPress={() =>
                  approve.mutate(code, { onSuccess: () => setDecided("approved") })
                }
              />
              <PrimaryButton
                title={strings.approveDevice.deny}
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
