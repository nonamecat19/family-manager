import { TELEGRAM_PROVIDER, toDisplayError, useIdentities, useUnlinkIdentity } from "@fm/api";
import type { Identity } from "@fm/sdk/auth/v1/auth_pb";
import { useTheme } from "@fm/ui";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ActivityIndicator, ScrollView, Text, View } from "react-native";

import { longDate, strings } from "../../components/i18n/index.ts";
import { EmptyState, Icon, IconButton, Screen, nocturne } from "../../components/nocturne/index.ts";
import { ActionSheet } from "./_layout.tsx";

function providerLabel(provider: string): string {
  if (provider === TELEGRAM_PROVIDER) return strings.connectedAccounts.providerTelegram;
  return provider;
}

export default function ConnectedAccountsScreen() {
  const router = useRouter();
  const identities = useIdentities();
  const unlink = useUnlinkIdentity();
  const theme = useTheme();
  const [unlinking, setUnlinking] = useState<Identity | null>(null);

  const rows = identities.data ?? [];
  const unlinkError = unlink.isError
    ? toDisplayError(unlink.error, strings.connectedAccounts.unlinkFailed).message
    : null;

  return (
    <Screen>
      <View className="flex-row items-center gap-[10px] px-[16px] pb-[10px] pt-[6px]">
        <IconButton
          icon="caret-left"
          label={strings.connectedAccounts.back}
          size={20}
          color={nocturne.accent.DEFAULT}
          onPress={() => router.back()}
        />
        <Text className="font-med text-[17px] text-fg">{strings.connectedAccounts.title}</Text>
      </View>

      {unlinkError ? (
        <Text
          className="px-[20px] pb-[8px] font-sans text-[12px]"
          style={{ color: theme.danger }}
        >
          {unlinkError}
        </Text>
      ) : null}

      {identities.isPending ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator color={nocturne.accent.DEFAULT} />
        </View>
      ) : identities.isError ? (
        <View className="flex-1 items-center justify-center px-[22px]">
          <Text className="text-center font-sans text-[13px] text-neutral-500">
            {toDisplayError(identities.error, strings.connectedAccounts.loadFailed).message}
          </Text>
        </View>
      ) : rows.length === 0 ? (
        <EmptyState
          title={strings.connectedAccounts.emptyTitle}
          body={strings.connectedAccounts.emptyBody}
          icon="link-simple"
          className="flex-1"
        />
      ) : (
        <ScrollView contentContainerClassName="gap-[10px] px-[20px] py-[12px]">
          {rows.map((identity) => (
            <View
              key={`${identity.provider}:${identity.externalId}`}
              className="flex-row items-center gap-[12px] rounded-md border border-neutral-800 bg-surface px-[14px] py-[12px]"
            >
              <View className="h-[34px] w-[34px] items-center justify-center rounded-full bg-accent-900">
                <Icon name="link-simple" size={16} color={nocturne.accent[400]} />
              </View>
              <View className="flex-1 gap-[2px]">
                <Text className="font-med text-[14px] text-fg">{providerLabel(identity.provider)}</Text>
                <Text className="font-sans text-[11.5px] text-neutral-600">{identity.externalId}</Text>
                {identity.linkedAt ? (
                  <Text className="font-sans text-[11px] text-neutral-600">
                    {strings.connectedAccounts.linked(longDate(identity.linkedAt))}
                  </Text>
                ) : null}
              </View>
              <IconButton
                icon="trash"
                label={strings.connectedAccounts.unlink}
                size={16}
                color={nocturne.neutral[500]}
                disabled={unlink.isPending}
                onPress={() => setUnlinking(identity)}
              />
            </View>
          ))}
        </ScrollView>
      )}

      <ActionSheet
        visible={unlinking !== null}
        title={strings.connectedAccounts.unlinkConfirm}
        onClose={() => setUnlinking(null)}
        actions={[
          {
            key: "confirm",
            label: strings.connectedAccounts.unlink,
            icon: "trash",
            tone: "danger",
            onPress: () => {
              if (!unlinking) return;
              unlink.mutate({ provider: unlinking.provider, externalId: unlinking.externalId });
            },
          },
        ]}
      />
    </Screen>
  );
}
