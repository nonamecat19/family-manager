import { parseAmount, useAccounts, useCreateAccount } from "@fm/api";
import { AccountType } from "@fm/sdk/finance/v1/finance_pb";
import { Button, Card, EmptyState, ErrorState, Field, Loading } from "@fm/ui";
import { useState } from "react";
import { Modal, Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { Amount } from "../../components/Amount";

const TYPES: readonly { label: string; value: AccountType }[] = [
  { label: "Cash", value: AccountType.CASH },
  { label: "Card", value: AccountType.CARD },
  { label: "Bank", value: AccountType.BANK },
  { label: "Savings", value: AccountType.SAVINGS },
  { label: "Debt", value: AccountType.DEBT },
];

export default function AccountsScreen() {
  const accounts = useAccounts();
  const [creating, setCreating] = useState(false);

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <ScrollView contentContainerClassName="gap-md p-lg pb-2xl">
        <View className="flex-row items-center justify-between">
          <Text className="text-title font-semibold text-fg dark:text-fg-dark">Accounts</Text>
          <Amount value={accounts.data?.total} size="title" />
        </View>

        {accounts.isPending ? (
          <Loading />
        ) : accounts.isError ? (
          <ErrorState message={accounts.error.message} onRetry={() => void accounts.refetch()} />
        ) : accounts.data.accounts.length === 0 ? (
          <EmptyState
            title="No accounts yet"
            hint="Add the wallet or card you spend from."
            action={<Button title="Add account" onPress={() => setCreating(true)} />}
          />
        ) : (
          accounts.data.accounts.map((a) => (
            <Card key={a.id} className="flex-row items-center justify-between p-lg">
              <View>
                <Text className="text-body text-fg dark:text-fg-dark">{a.name}</Text>
                <Text className="text-caption text-muted dark:text-muted-dark">
                  {a.currencyCode}
                </Text>
              </View>
              <Amount value={a.balance} />
            </Card>
          ))
        )}

        {accounts.data && accounts.data.accounts.length > 0 ? (
          <Button title="Add account" variant="secondary" onPress={() => setCreating(true)} />
        ) : null}
      </ScrollView>

      <NewAccountModal visible={creating} onClose={() => setCreating(false)} />
    </SafeAreaView>
  );
}

function NewAccountModal({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const createAccount = useCreateAccount();

  const [name, setName] = useState("");
  const [currency, setCurrency] = useState("EUR");
  const [opening, setOpening] = useState("0");
  const [type, setType] = useState<AccountType>(AccountType.CASH);

  const parsed = parseAmount(opening, currency);
  const canSubmit = name.trim() !== "" && currency.length === 3 && parsed !== null;

  const submit = () => {
    if (!parsed) return;
    createAccount.mutate(
      {
        name: name.trim(),
        type,
        currencyCode: currency.toUpperCase(),
        openingBalance: parsed,
        color: "",
        icon: "",
      },
      {
        onSuccess: () => {
          setName("");
          setOpening("0");
          onClose();
        },
      },
    );
  };

  return (
    <Modal visible={visible} animationType="slide" onRequestClose={onClose} transparent={false}>
      <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
        <ScrollView contentContainerClassName="gap-lg p-xl">
          <Text className="text-title font-semibold text-fg dark:text-fg-dark">New account</Text>

          <Field label="Name" value={name} onChangeText={setName} placeholder="Wallet" />
          <Field
            label="Currency"
            value={currency}
            onChangeText={(t) => setCurrency(t.toUpperCase().slice(0, 3))}
            autoCapitalize="characters"
            maxLength={3}
          />
          <Field
            label="Opening balance"
            value={opening}
            onChangeText={setOpening}
            keyboardType="decimal-pad"
            error={parsed === null ? "Enter a number" : undefined}
          />

          <View className="gap-xs">
            <Text className="text-caption text-muted dark:text-muted-dark">Type</Text>
            <View className="flex-row flex-wrap gap-xs">
              {TYPES.map((t) => (
                <Pressable
                  key={t.value}
                  accessibilityRole="button"
                  accessibilityState={{ selected: t.value === type }}
                  onPress={() => setType(t.value)}
                  className={`rounded-md px-md py-xs ${
                    t.value === type ? "bg-primary" : "border border-border dark:border-border-dark"
                  }`}
                >
                  <Text
                    className={`text-caption ${
                      t.value === type ? "text-primary-fg" : "text-muted dark:text-muted-dark"
                    }`}
                  >
                    {t.label}
                  </Text>
                </Pressable>
              ))}
            </View>
          </View>

          {createAccount.isError ? (
            <Text className="text-caption text-expense">{createAccount.error.message}</Text>
          ) : null}

          <Button
            title="Create"
            loading={createAccount.isPending}
            disabled={!canSubmit}
            onPress={submit}
          />
          <Button title="Cancel" variant="secondary" onPress={onClose} />
        </ScrollView>
      </SafeAreaView>
    </Modal>
  );
}
