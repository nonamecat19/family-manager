import {
  fromWire,
  toDisplayError,
  useAccounts,
  useFamily,
  useFinanceMembers,
  type Account,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  Button,
  Drawer,
  EmptyState,
  Fab,
  Icon,
  Kicker,
  MoneyText,
  Screen,
  ScreenHeader,
  formatMoney,
  nocturne,
  useDrawerItems,
} from "@/components/nocturne";
import { AccountRow, HiddenPrivateRow } from "@/components/screens/accounts/AccountRow.tsx";
import { NewAccountSheet } from "@/components/screens/accounts/NewAccountSheet.tsx";
import { TransferSheet } from "@/components/screens/accounts/TransferSheet.tsx";

/**
 * Screen 08 — Accounts.
 *
 * The screen's one idea: shared money and private money are drawn as two different things.
 * The headline counts the shared accounts only, private accounts are captioned with why they
 * are missing from it, and another member's private accounts appear as a count with no
 * balance at all.
 *
 * No filtering happens here. `ListAccounts` returns the split already made — shared,
 * private_own, hidden — because the server decides what the caller may see, so a bug in this
 * file cannot leak a balance the API never sent.
 */
export default function AccountsScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const accounts = useAccounts();
  const members = useFinanceMembers();
  const family = useFamily();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [transferOpen, setTransferOpen] = useState(false);
  const [newAccountOpen, setNewAccountOpen] = useState(false);

  const drawerItems = useDrawerItems();

  const data = accounts.data;
  const shared = data?.shared ?? [];
  const privateOwn = data?.privateOwn ?? [];
  const hidden = data?.hidden ?? [];

  const sharedBalance = fromWire(data?.sharedBalance);
  const savings = fromWire(data?.savingsTotal, sharedBalance.currencyCode);

  // Whose private accounts these are. The contract lets a member create a private account only
  // for themselves, so the owner of `privateOwn` IS the signed-in member — which is also the
  // only way this app can name the caller: no RPC marks a member as "me".
  const selfId = privateOwn[0]?.ownerMemberId ?? "";
  const self = members.data?.find((m) => m.userId === selfId);
  const selfName = self?.displayName ?? "";

  // What a transfer may move between: shared plus the caller's own private accounts, never a
  // hidden one — those are only ever a count.
  const transferable: Account[] = useMemo(() => [...shared, ...privateOwn], [shared, privateOwn]);

  const isEmpty = shared.length === 0 && privateOwn.length === 0 && hidden.length === 0;

  return (
    <Screen>
      <ScreenHeader
        gradient
        title={t("accounts.title")}
        leading={{ icon: "list", label: t("nav.menu"), onPress: () => setDrawerOpen(true) }}
      >
        <View className="items-center pt-n3">
          <Text className="text-[11px] text-neutral-500">{t("accounts.available")}</Text>
          <MoneyText value={sharedBalance} size={27} weight="medium" className="mt-[2px]" />
          {savings.amountMinor !== 0 ? (
            <Text className="mt-[2px] text-[11px] text-neutral-500">
              {t("accounts.inSavings", { amount: formatMoney(savings) })}
            </Text>
          ) : null}

          <View className="mt-n4 flex-row justify-center gap-n3">
            <Button
              title={t("accounts.history")}
              variant="ghost"
              icon="clock-counter-clockwise"
              onPress={() => router.push("/(app)/transactions")}
            />
            <Button
              title={t("accounts.transfer")}
              variant="ghost"
              icon="arrows-left-right"
              onPress={() => setTransferOpen(true)}
            />
          </View>
        </View>
      </ScreenHeader>

      {accounts.isPending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
        </View>
      ) : accounts.isError ? (
        <LoadError error={accounts.error} onRetry={() => void accounts.refetch()} />
      ) : isEmpty ? (
        <View className="flex-1 justify-center">
          <EmptyState
            icon="wallet"
            title={t("accounts.emptyTitle")}
            body={t("accounts.emptyBody")}
            action={{ label: t("accounts.addAccount"), onPress: () => setNewAccountOpen(true) }}
          />
        </View>
      ) : (
        <ScrollView
          className="flex-1"
          contentContainerStyle={{
            paddingHorizontal: nocturne.space.n4,
            paddingTop: nocturne.space.n5,
            // Clears the Fab, which floats over the list rather than reserving space.
            paddingBottom: 96,
            gap: nocturne.space.n3,
          }}
          showsVerticalScrollIndicator={false}
        >
          {shared.length > 0 ? (
            <View className="mb-n1 flex-row items-baseline justify-between px-n1">
              <Kicker>{t("accounts.sharedSection")}</Kicker>
              <Text className="text-[10.5px] text-neutral-600">{t("accounts.visibleToAll")}</Text>
            </View>
          ) : null}
          {shared.map((account) => (
            <AccountRow key={account.id} account={account} />
          ))}

          {privateOwn.length > 0 ? (
            <View className="mb-n1 mt-n4 flex-row items-center justify-between px-n1">
              <Kicker>{t("accounts.privateSection", { name: selfName }).trim()}</Kicker>
              <Icon name="eye-slash" size={14} color={nocturne.neutral[600]} />
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

      <Drawer
        visible={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        account={{ name: selfName || (family.data?.family?.name ?? ""), email: self?.email ?? "" }}
        household={
          family.data?.family
            ? { name: family.data.family.name, balance: sharedBalance }
            : undefined
        }
        items={drawerItems}
        activeId="accounts"
        onSelect={(item) => {
          setDrawerOpen(false);
          if (item.href && item.id !== "accounts") router.push(item.href);
        }}
      />
    </Screen>
  );
}

/** The list's own failure, drawn like the gate's: what happened, its reference, and a retry. */
function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const { t } = useI18n();
  const shown = toDisplayError(error, t("common.loadFailed"));
  return (
    <View className="flex-1 justify-center gap-n4 px-n6">
      <Text className="text-[13.5px] leading-[21px] text-neutral-500">{shown.message}</Text>
      {shown.reference ? (
        <Text className="text-[12px] text-neutral-600">
          {t("common.errorReference", { ref: shown.reference })}
        </Text>
      ) : null}
      <Button title={t("common.tryAgain")} onPress={onRetry} />
    </View>
  );
}
