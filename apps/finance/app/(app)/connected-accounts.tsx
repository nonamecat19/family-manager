import { TELEGRAM_PROVIDER, toDisplayError, useIdentities, useUnlinkIdentity } from "@fm/api";
import type { Identity } from "@fm/sdk/auth/v1/auth_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Text, View } from "react-native";

import { useI18n, type TranslationKey } from "@/components/i18n";
import {
  Button,
  EmptyState,
  IconCircle,
  Row,
  Screen,
  ScreenHeader,
  Sheet,
  nocturne,
} from "@/components/nocturne";

function providerLabel(provider: string, t: (key: TranslationKey) => string): string {
  if (provider === TELEGRAM_PROVIDER) return t("connectedAccounts.providerTelegram");
  return provider;
}

function linkedDate(linkedAt: Identity["linkedAt"]): string {
  if (!linkedAt) return "";
  const date = new Date(Number(linkedAt.seconds) * 1000 + Math.floor(linkedAt.nanos / 1e6));
  return date.toLocaleDateString();
}

export default function ConnectedAccountsScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const identities = useIdentities();
  const unlink = useUnlinkIdentity();
  const [unlinking, setUnlinking] = useState<Identity | null>(null);

  const rows = identities.data ?? [];
  const unlinkError = unlink.isError
    ? toDisplayError(unlink.error, t("connectedAccounts.unlinkFailed")).message
    : null;

  return (
    <Screen>
      <ScreenHeader
        title={t("connectedAccounts.title")}
        leading={{ icon: "arrow-left", label: t("common.back"), onPress: () => router.back() }}
      />

      {unlinkError ? (
        <Text className="px-n5 pb-n3 text-[12px] text-overspend">{unlinkError}</Text>
      ) : null}

      {identities.isPending ? (
        <Text className="px-n5 text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
      ) : identities.isError ? (
        <EmptyState
          icon="device-mobile"
          title={t("gate.errorTitle")}
          body={toDisplayError(identities.error, t("connectedAccounts.loadFailed")).message}
          action={{ label: t("common.tryAgain"), onPress: () => void identities.refetch() }}
        />
      ) : rows.length === 0 ? (
        <EmptyState
          icon="device-mobile"
          title={t("connectedAccounts.emptyTitle")}
          body={t("connectedAccounts.emptyBody")}
        />
      ) : (
        <View>
          {rows.map((identity) => (
            <Row
              key={`${identity.provider}:${identity.externalId}`}
              title={providerLabel(identity.provider, t)}
              subtitle={
                identity.linkedAt
                  ? `${identity.externalId} · ${t("connectedAccounts.linked", { date: linkedDate(identity.linkedAt) })}`
                  : identity.externalId
              }
              leading={
                <IconCircle
                  icon="device-mobile"
                  size={34}
                  tint={{ bg: nocturne.neutral[800], fg: nocturne.accent[400] }}
                />
              }
              trailing={
                <Button
                  title={t("connectedAccounts.unlink")}
                  variant="ghost"
                  disabled={unlink.isPending}
                  onPress={() => setUnlinking(identity)}
                />
              }
            />
          ))}
        </View>
      )}

      <Sheet
        visible={unlinking !== null}
        onClose={() => setUnlinking(null)}
        title={t("connectedAccounts.unlinkConfirm")}
      >
        <View className="gap-n3 pb-n4">
          <Button
            title={t("connectedAccounts.unlink")}
            disabled={unlink.isPending}
            onPress={() => {
              if (!unlinking) return;
              unlink.mutate(
                { provider: unlinking.provider, externalId: unlinking.externalId },
                { onSuccess: () => setUnlinking(null) },
              );
            }}
          />
          <Button title={t("common.cancel")} variant="ghost" onPress={() => setUnlinking(null)} />
        </View>
      </Sheet>
    </Screen>
  );
}
