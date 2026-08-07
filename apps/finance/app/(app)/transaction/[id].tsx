import { fromWire, useClients, useDeleteTransaction, useUpdateTransaction } from "@fm/api";
import { queryKeys } from "@fm/api";
import { useQuery } from "@tanstack/react-query";
import { ErrorState, Loading } from "@fm/ui";
import { useLocalSearchParams, useRouter } from "expo-router";
import { Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { TransactionForm } from "../../../components/TransactionForm";

export default function EditTransactionScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { finance } = useClients();

  const transaction = useQuery({
    queryKey: queryKeys.transaction(id),
    queryFn: () => finance.getTransaction({ id }),
    enabled: id !== undefined && id !== "",
  });

  const update = useUpdateTransaction();
  const remove = useDeleteTransaction();

  if (transaction.isPending) return <Loading />;
  if (transaction.isError) {
    return (
      <ErrorState
        message={transaction.error.message}
        onRetry={() => void transaction.refetch()}
      />
    );
  }

  const tx = transaction.data.transaction;
  if (!tx) return <ErrorState message="That transaction no longer exists." />;

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <View className="px-xl pt-lg">
        <Text className="text-title font-semibold text-fg dark:text-fg-dark">Edit transaction</Text>
      </View>
      <TransactionForm
        initial={{
          accountId: tx.accountId,
          counterAccountId: tx.counterAccountId,
          categoryId: tx.categoryId,
          type: tx.type,
          amount: fromWire(tx.amount),
          note: tx.note,
          occurredOn: tx.occurredOn,
        }}
        submitLabel="Save"
        submitting={update.isPending}
        error={update.isError ? update.error.message : undefined}
        onSubmit={(input) =>
          update.mutate({ ...input, id: tx.id }, { onSuccess: () => router.back() })
        }
        onDelete={() => remove.mutate(tx.id, { onSuccess: () => router.back() })}
      />
    </SafeAreaView>
  );
}
