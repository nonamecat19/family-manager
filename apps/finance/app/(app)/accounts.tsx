import { LoadError } from "@/components/kit/LoadError.tsx";
import {
  fromWire,
  useAccounts,
  useFinanceMembers,
  type Account,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { Fab, formatMoney, MoneyText } from "@/components/kit";
import { AccountRow, HiddenPrivateRow } from "@/components/screens/accounts/AccountRow.tsx";
import { NewAccountSheet } from "@/components/screens/accounts/NewAccountSheet.tsx";
import { TransferSheet } from "@/components/screens/accounts/TransferSheet.tsx";
import { Button, EmptyState, Icon, Kicker, Screen, ScreenHeader, organic } from "@fm/ui";

export default function AccountsScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const accounts = useAccounts();
  const members = useFinanceMembers();

  const [transferOpen, setTransferOpen] = useState(false);
  const [newAccountOpen, setNewAccountOpen] = useState(false);

  const data = accounts.data;
  const shared = data?.shared ?? [];
  const privateOwn = data?.privateOwn ?? [];
  const hidden = data?.hidden ?? [];

  const sharedBalance = fromWire(data?.sharedBalance);
  const savings = fromWire(data?.savingsTotal, sharedBalance.currencyCode);

  const selfId = privateOwn[0]?.ownerMemberId ?? "";
  const self = members.data?.find((m) => m.userId === selfId);
  const selfName = self?.displayName ?? "";

  const transferable: Account[] = useMemo(() => [...shared, ...privateOwn], [shared, privateOwn]);

  const isEmpty = shared.length === 0 && privateOwn.length === 0 && hidden.length === 0;

  return (
    <Screen>
      <View className="gap-5 px-5.5 pt-sm">
        <ScreenHeader title={t("accounts.title")} />

        <View className="items-center">
          <Text className="text-11 text-neutral-600">{t("accounts.available")}</Text>
          <MoneyText value={sharedBalance} size={27} weight="medium" className="mt-0.5" />
          {savings.amountMinor !== 0 ? (
            <Text className="mt-0.5 text-11 text-neutral-600">
              {t("accounts.inSavings", { amount: formatMoney(savings) })}
            </Text>
          ) : null}

          <View className="mt-lg flex-row justify-center gap-3">
            <Button
              title={t("accounts.history")}
              tone="quiet"
              onPress={() => router.push("/(app)/transactions")}
            />
            <Button
              title={t("accounts.transfer")}
              tone="quiet"
              onPress={() => setTransferOpen(true)}
            />
          </View>
        </View>
      </View>

      {accounts.isPending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-13 text-neutral-600">{t("common.loadingEllipsis")}</Text>
        </View>
      ) : accounts.isError ? (
        <LoadError error={accounts.error} onRetry={() => void accounts.refetch()} />
      ) : isEmpty ? (
        <View className="flex-1 justify-center">
          <EmptyState
            title={t("accounts.emptyTitle")}
            body={t("accounts.emptyBody")}
            action={{ label: t("accounts.addAccount"), onPress: () => setNewAccountOpen(true) }}
          />
        </View>
      ) : (
        <ScrollView
          className="flex-1"
          contentContainerStyle={{
            paddingHorizontal: 11.2,
            paddingTop: 16.8,
            paddingBottom: 96,
            gap: 8.4,
          }}
          showsVerticalScrollIndicator={false}
        >
          {shared.length > 0 ? (
            <View className="mb-0.7 flex-row items-baseline justify-between px-0.7">
              <Kicker>{t("accounts.sharedSection")}</Kicker>
              <Text className="text-10.5 text-neutral-600">{t("accounts.visibleToAll")}</Text>
            </View>
          ) : null}
          {shared.map((account) => (
            <AccountRow key={account.id} account={account} />
          ))}

          {privateOwn.length > 0 ? (
            <View className="mb-0.7 mt-2.8 flex-row items-center justify-between px-0.7">
              <Kicker>{t("accounts.privateSection", { name: selfName }).trim()}</Kicker>
              <Icon name="eye-slash" size={14} color={organic.neutral[600]} />
            </View>
          ) : null}
          {privateOwn.map((account) => (
            <AccountRow key={account.id} account={account} tone="private" />
          ))}

          {hidden.map((summary) => (
            <HiddenPrivateRow key={summary.memberId} summary={summary} />
          ))}
        </ScrollView>
      )}

      <Fab label={t("accounts.addAccount")} onPress={() => setNewAccountOpen(true)} />

      <TransferSheet
        visible={transferOpen}
        onClose={() => setTransferOpen(false)}
        accounts={transferable}
      />
      <NewAccountSheet
        visible={newAccountOpen}
        onClose={() => setNewAccountOpen(false)}
        currencyCode={sharedBalance.currencyCode}
      />
    </Screen>
  );
}
