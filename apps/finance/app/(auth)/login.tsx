import { useClients } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import { Button, Field } from "@fm/ui";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

export default function LoginScreen() {
  const { auth } = useClients();
  const { signIn } = useAuth();

  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    setError(null);
    setBusy(true);
    try {
      if (mode === "register") {
        await auth.register({ email, password, name });
      }
      const res = await auth.login({ email, password });
      await signIn(tokensFromResponse(res, Date.now()));
    } catch {
      // Never echo the server's message verbatim on an auth screen: it distinguishes
      // "no such user" from "wrong password", which is an account-enumeration oracle.
      setError(
        mode === "register"
          ? "Could not create that account. Try a different email."
          : "Email or password is incorrect.",
      );
    } finally {
      setBusy(false);
    }
  };

  const canSubmit = email.trim() !== "" && password !== "" && !busy;

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        className="flex-1 justify-center gap-lg p-xl"
      >
        <View className="gap-xs">
          <Text className="text-display font-bold text-fg dark:text-fg-dark">Family Finance</Text>
          <Text className="text-body text-muted dark:text-muted-dark">
            {mode === "login" ? "Sign in to your household ledger." : "Create your account."}
          </Text>
        </View>

        {mode === "register" ? (
          <Field label="Name" value={name} onChangeText={setName} autoComplete="name" />
        ) : null}

        <Field
          label="Email"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
        />
        <Field
          label="Password"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          error={error ?? undefined}
        />

        <Button
          title={mode === "login" ? "Sign in" : "Create account"}
          loading={busy}
          disabled={!canSubmit}
          onPress={() => void submit()}
        />

        <Pressable
          accessibilityRole="button"
          onPress={() => {
            setMode(mode === "login" ? "register" : "login");
            setError(null);
          }}
        >
          <Text className="text-center text-body text-primary">
            {mode === "login" ? "Create an account" : "I already have an account"}
          </Text>
        </Pressable>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
