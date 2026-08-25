import { useClients } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, Text, View } from "react-native";

import { Display, Field, PrimaryButton, Screen } from "../../components/organic/ui.tsx";

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
    <Screen>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        className="flex-1 justify-center gap-[18px] px-[24px]"
      >
        <View>
          <Display size={36}>Family Recipes</Display>
          <Text className="mt-[10px] font-fig text-[15.5px] leading-[23px] text-neutral-700">
            {mode === "login"
              ? "Sign in to your family cookbook."
              : "Create your account and start writing it down."}
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

        <PrimaryButton
          title={busy ? "One moment…" : mode === "login" ? "Sign in" : "Create account"}
          disabled={!canSubmit}
          onPress={() => void submit()}
        />

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={mode === "login" ? "Create an account" : "I already have an account"}
          onPress={() => {
            setMode(mode === "login" ? "register" : "login");
            setError(null);
          }}
        >
          <Text className="text-center font-fig-bold text-[14.5px] text-accent-700">
            {mode === "login" ? "Create an account" : "I already have an account"}
          </Text>
        </Pressable>
      </KeyboardAvoidingView>
    </Screen>
  );
}
