import { TELEGRAM_PROVIDER, toDisplayError, useIdentities, useUnlinkIdentity } from "@fm/api";
import type { Identity } from "@fm/sdk/auth/v1/auth_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ActivityIndicator, Text, View } from "react-native";

import { longDate, strings } from "../../components/i18n/index.ts";
import { EmptyState, Icon, RoundButton, Screen, ScreenHeader, ScrollBody, organic } from "@fm/ui";
import { ActionSheet } from "./_layout.tsx";

function providerLabel(provider: string): string {
  if (provider === TELEGRAM_PROVIDER) return strings.connectedAccounts.providerTelegram;
  return provider;
}

export default function ConnectedAccountsScreen() {
  const router = useRouter();
  const identities = useIdentities();
  const unlink = useUnlinkIdentity();
  const [unlinking, setUnlinking] = useState<Identity | null>(null);

  const rows = identities.data ?? [];
  const unlinkError = unlink.isError
    ? toDisplayError(unlink.error, strings.connectedAccounts.unlinkFailed).message
    : null;

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader
          title={strings.connectedAccounts.title}
          onBack={() => router.back()}
          backLabel={strings.connectedAccounts.back}
        />

        {unlinkError ? (
          <Text className="font-fig-semi text-12" style={{ color: organic.danger }}>
            {unlinkError}
          </Text>
        ) : null}

        {identities.isPending ? (
          <View className="items-center py-7">
            <ActivityIndicator color={organic.accent.DEFAULT} />
          </View>
        ) : identities.isError ? (
          <Text className="font-fig text-14 leading-[21px] text-neutral-600">
            {toDisplayError(identities.error, strings.connectedAccounts.loadFailed).message}
          </Text>
        ) : rows.length === 0 ? (
          <EmptyState
            title={strings.connectedAccounts.emptyTitle}
            body={strings.connectedAccounts.emptyBody}
          />
        ) : (
          <View className="gap-2.5">
            {rows.map((identity) => (
              <View
                key={`${identity.provider}:${identity.externalId}`}
                className="flex-row items-center gap-3 rounded-2xl bg-neutral-100 px-lg py-3.5"
              >
                <View className="h-[34px] w-[34px] items-center justify-center rounded-full bg-accent-100">
                  <Icon name="link-simple" size={16} color={organic.accent[600]} />
                </View>
                <View className="flex-1 gap-0.75">
                  <Text className="font-fig-bold text-15.5 text-fg">{providerLabel(identity.provider)}</Text>
                  <Text className="font-fig text-12.5 text-neutral-600">{identity.externalId}</Text>
                  {identity.linkedAt ? (
                    <Text className="font-fig text-12 text-neutral-600">
                      {strings.connectedAccounts.linked(longDate(identity.linkedAt))}
                    </Text>
                  ) : null}
                </View>
                <RoundButton
                  icon="trash"
                  label={strings.connectedAccounts.unlink}
                  disabled={unlink.isPending}
                  onPress={() => setUnlinking(identity)}
                />
              </View>
            ))}
          </View>
        )}
      </ScrollBody>

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
