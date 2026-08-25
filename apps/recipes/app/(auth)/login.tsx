import { useClients } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, Text, View } from "react-native";

import { useI18n } from "../../components/i18n/index.tsx";
import { Display, Field, PrimaryButton, Screen } from "../../components/organic/ui.tsx";

export default function LoginScreen() {
  const { auth } = useClients();
  const { signIn } = useAuth();
  const { t } = useI18n();

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
      setError(mode === "register" ? t("login.registerError") : t("login.loginError"));
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
          <Display size={36}>{t("login.appName")}</Display>
          <Text className="mt-[10px] font-fig text-[15.5px] leading-[23px] text-neutral-700">
            {mode === "login" ? t("login.signInBody") : t("login.registerBody")}
          </Text>
        </View>

        {mode === "register" ? (
          <Field label={t("login.name")} value={name} onChangeText={setName} autoComplete="name" />
        ) : null}

        <Field
          label={t("login.email")}
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
        />
        <Field
          label={t("login.password")}
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          error={error ?? undefined}
        />

        <PrimaryButton
          title={busy ? t("login.oneMoment") : mode === "login" ? t("login.signIn") : t("login.createAccount")}
          disabled={!canSubmit}
          onPress={() => void submit()}
        />

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={mode === "login" ? t("login.switchToRegister") : t("login.switchToLogin")}
          onPress={() => {
            setMode(mode === "login" ? "register" : "login");
            setError(null);
          }}
        >
          <Text className="text-center font-fig-bold text-[14.5px] text-accent-700">
            {mode === "login" ? t("login.switchToRegister") : t("login.switchToLogin")}
          </Text>
        </Pressable>
      </KeyboardAvoidingView>
    </Screen>
  );
}
