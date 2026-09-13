import { toDisplayError, useClients } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, View } from "react-native";

import { Body, Button, Caption, Field, Screen, TextLink } from "@fm/ui";

import { useI18n } from "../../components/i18n/index.tsx";
import { Display } from "../../components/organic/ui.tsx";

export default function LoginScreen() {
  const { auth } = useClients();
  const { signIn } = useAuth();
  const { t } = useI18n();

  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [errorRef, setErrorRef] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    setError(null);
    setErrorRef(null);
    setBusy(true);
    try {
      if (mode === "register") {
        await auth.register({ email, password, name });
      }
      const res = await auth.login({ email, password });
      await signIn(tokensFromResponse(res, Date.now()));
    } catch (e) {
      const shown = toDisplayError(
        e,
        mode === "register" ? t("login.registerError") : t("login.loginError"),
      );
      setError(shown.message);
      setErrorRef(shown.reference ?? null);
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
          {
}
          <Display size={36}>{t("login.appName")}</Display>
          <View className="mt-[10px]">
            <Body>{mode === "login" ? t("login.signInBody") : t("login.registerBody")}</Body>
          </View>
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

        {errorRef ? <Caption>{t("common.errorReference", { ref: errorRef })}</Caption> : null}

        <Button
          title={mode === "login" ? t("login.signIn") : t("login.createAccount")}
          disabled={!canSubmit}
          busy={busy}
          onPress={() => void submit()}
        />

        <TextLink
          label={mode === "login" ? t("login.switchToRegister") : t("login.switchToLogin")}
          onPress={() => {
            setMode(mode === "login" ? "register" : "login");
            setError(null);
          }}
        />
      </KeyboardAvoidingView>
    </Screen>
  );
}
