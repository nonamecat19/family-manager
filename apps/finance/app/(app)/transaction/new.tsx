import { useCreateTransaction } from "@fm/api";
import { useRouter } from "expo-router";
import { Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { TransactionForm } from "../../../components/TransactionForm";

export default function NewTransactionScreen() {
  const router = useRouter();
  const create = useCreateTransaction();

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="px-xl pt-lg">
        <Text className="text-title font-semibold text-fg dark:text-fg-dark">New transaction</Text>
      </View>
      <TransactionForm
        submitLabel="Add"
        submitting={create.isPending}
        error={create.isError ? create.error.message : undefined}
        onSubmit={(input) => create.mutate(input, { onSuccess: () => router.back() })}
      />
    </SafeAreaView>
  );
}
