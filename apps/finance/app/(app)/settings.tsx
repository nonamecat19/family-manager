import { toDisplayError, useCategories, useCreateCategory, useFamily, useInviteMember, useMembers } from "@fm/api";
import { useAuth } from "@fm/auth";
import { TransactionType } from "@fm/sdk/finance/v1/finance_pb";
import { categoryColor, categoryPalette } from "@fm/theme";
import { Button, Card, Dot, ErrorState, Field, Loading } from "@fm/ui";
import { useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

export default function SettingsScreen() {
  const { signOut } = useAuth();
  const family = useFamily();
  const familyId = family.data?.family?.id ?? "";
  const members = useMembers(familyId);

  if (family.isPending) return <Loading />;
  if (family.isError) {
    return <ErrorState
            {...toDisplayError(family.error, "Could not load your household.")}
            onRetry={() => void family.refetch()}
          />;
  }

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <ScrollView contentContainerClassName="gap-xl p-lg pb-2xl">
        <Section title="Household">
          <Card className="gap-xs p-lg">
            <Text className="text-body text-fg dark:text-fg-dark">
              {family.data.family?.name}
            </Text>
            <Text className="text-caption text-muted dark:text-muted-dark">
              {members.data?.members.length ?? 0} member
              {(members.data?.members.length ?? 0) === 1 ? "" : "s"}
            </Text>
          </Card>
          {(members.data?.members ?? []).map((m) => (
            <Card key={m.userId} className="flex-row justify-between p-md">
              <Text className="text-body text-fg dark:text-fg-dark">
                {m.displayName !== "" ? m.displayName : m.email}
              </Text>
              <Text className="text-caption text-muted dark:text-muted-dark">
                {m.role === 1 ? "admin" : "member"}
              </Text>
            </Card>
          ))}
          <InviteBox familyId={familyId} />
        </Section>

        <Section title="Categories">
          <CategoryManager />
        </Section>

        <Button title="Sign out" variant="secondary" onPress={() => void signOut()} />
      </ScrollView>
    </SafeAreaView>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View className="gap-sm">
      <Text className="text-title font-semibold text-fg dark:text-fg-dark">{title}</Text>
      {children}
    </View>
  );
}

function InviteBox({ familyId }: { familyId: string }) {
  const invite = useInviteMember();
  const [email, setEmail] = useState("");

  return (
    <View className="gap-sm">
      <Field
        label="Invite by email"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        keyboardType="email-address"
      />
      <Button
        title="Create invitation"
        variant="secondary"
        loading={invite.isPending}
        disabled={email.trim() === "" || familyId === ""}
        onPress={() => invite.mutate({ familyId, email: email.trim() })}
      />
      {invite.data ? (
        <Card className="gap-xs p-md">
          <Text className="text-caption text-muted dark:text-muted-dark">
            Share this code — it is shown once and expires.
          </Text>
          <Text selectable className="text-body text-fg dark:text-fg-dark">
            {invite.data.token}
          </Text>
        </Card>
      ) : null}
      {invite.isError ? (
        <Text className="text-caption text-expense">{invite.error.message}</Text>
      ) : null}
    </View>
  );
}

function CategoryManager() {
  const categories = useCategories();
  const create = useCreateCategory();

  const [name, setName] = useState("");
  const [kind, setKind] = useState<TransactionType>(TransactionType.EXPENSE);

  return (
    <View className="gap-sm">
      {(categories.data?.categories ?? []).map((c, i) => (
        <Card key={c.id} className="flex-row items-center gap-md p-md">
          <Dot color={categoryColor(c.color, i, categoryPalette)} />
          <Text className="flex-1 text-body text-fg dark:text-fg-dark">{c.name}</Text>
          <Text className="text-caption text-muted dark:text-muted-dark">
            {c.kind === TransactionType.INCOME ? "income" : "expense"}
          </Text>
        </Card>
      ))}

      <Field label="New category" value={name} onChangeText={setName} placeholder="Groceries" />
      <View className="flex-row gap-xs">
        <KindToggle
          label="Expense"
          active={kind === TransactionType.EXPENSE}
          onPress={() => setKind(TransactionType.EXPENSE)}
        />
        <KindToggle
          label="Income"
          active={kind === TransactionType.INCOME}
          onPress={() => setKind(TransactionType.INCOME)}
        />
      </View>
      <Button
        title="Add category"
        variant="secondary"
        loading={create.isPending}
        disabled={name.trim() === ""}
        onPress={() =>
          create.mutate(
            {
              name: name.trim(),
              kind,
              // An empty colour lets the palette assign one, keeping charts consistent.
              color: "",
              icon: "",
            },
            { onSuccess: () => setName("") },
          )
        }
      />
      {create.isError ? (
        <Text className="text-caption text-expense">{create.error.message}</Text>
      ) : null}
    </View>
  );
}

function KindToggle({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected: active }}
      onPress={onPress}
      className={`flex-1 rounded-md py-xs ${
        active ? "bg-primary" : "border border-border dark:border-border-dark"
      }`}
    >
      <Text
        className={`text-center text-caption ${
          active ? "text-primary-fg" : "text-muted dark:text-muted-dark"
        }`}
      >
        {label}
      </Text>
    </Pressable>
  );
}
