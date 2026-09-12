import { toDisplayError, useClients } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, View } from "react-native";

import { Body, Button, Caption, Field, Heading, Screen, TextLink } from "@fm/ui";

import { useI18n } from "../../components/i18n/index.tsx";
import { Icon, nocturne } from "../../components/nocturne/index.ts";

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
      // The service phrases the useful failures itself — "that email is already registered",
      // "password must be at least 8 characters". toDisplayError keeps those and falls back
      // to the generic copy only for failures with nothing readable in them.
      const shown = toDisplayError(e, mode === "register" ? t("auth.registerError") : t("auth.loginError"));
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
        className="flex-1 justify-center gap-n5 px-n6"
      >
        <View>
          {/* The app's own mark stays app-local: a brand is the one thing a shared component
              set must not try to own. Everything below it is @fm/ui. */}
          <View className="mb-n5 h-[44px] w-[44px] items-center justify-center rounded-md border border-accent">
            <Icon name="wallet" size={22} color={nocturne.accent[400]} />
          </View>
          <Heading>{t("auth.title")}</Heading>
          <View className="mt-n3">
            <Body>{mode === "login" ? t("auth.signInBody") : t("auth.registerBody")}</Body>
          </View>
        </View>

        {mode === "register" ? (
          <Field label={t("auth.name")} value={name} onChangeText={setName} autoComplete="name" />
        ) : null}

        <Field
          label={t("auth.email")}
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
        />
        <Field
          label={t("auth.password")}
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          error={error ?? undefined}
        />

        {errorRef ? <Caption>{t("common.errorReference", { ref: errorRef })}</Caption> : null}

        <Button
          title={mode === "login" ? t("auth.signIn") : t("auth.createAccount")}
          disabled={!canSubmit}
          busy={busy}
          onPress={() => void submit()}
        />

        <TextLink
          label={mode === "login" ? t("auth.switchToRegister") : t("auth.switchToLogin")}
          onPress={() => {
            setMode(mode === "login" ? "register" : "login");
            setError(null);
          }}
        />
      </KeyboardAvoidingView>
    </Screen>
  );
}
