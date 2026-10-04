import { toDisplayError, useClients, useTelegramLogin } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import Constants from "expo-constants";
import * as Linking from "expo-linking";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Text, View } from "react-native";
import { Body, Button, Caption, Display, Field, Screen, TextLink } from "@fm/ui";

import { useI18n } from "../../components/i18n/index.tsx";

const telegramBot = (Constants.expoConfig?.extra as { telegramBot?: string } | undefined)?.telegramBot;

export default function LoginScreen() {
  const { auth } = useClients();
  const { signIn } = useAuth();
  const { t } = useI18n();
  const telegram = useTelegramLogin({ bot: telegramBot, open: Linking.openURL });

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

        {telegram.available ? <TelegramLogin telegram={telegram} /> : null}
      </KeyboardAvoidingView>
    </Screen>
  );
}

function TelegramLogin({ telegram }: { telegram: ReturnType<typeof useTelegramLogin> }) {
  const { t } = useI18n();

  if (telegram.phase === "pending" && telegram.userCode) {
    return (
      <View className="items-center gap-[10px]">
        <Text className="font-fig-x text-[22px] text-fg" style={{ letterSpacing: 3 }}>
          {telegram.userCode}
        </Text>
        <Caption>{t("login.telegramPendingHint")}</Caption>
        <TextLink label={t("login.telegramCancel")} onPress={telegram.cancel} />
      </View>
    );
  }

  const failure =
    telegram.phase === "denied"
      ? t("login.telegramDenied")
      : telegram.phase === "expired"
        ? t("login.telegramExpired")
        : telegram.phase === "error"
          ? toDisplayError(telegram.error, t("login.telegramError")).message
          : null;

  if (failure) {
    return (
      <View className="items-center gap-[10px]">
        <Caption>{failure}</Caption>
        <Button title={t("login.telegramRetry")} onPress={() => void telegram.start()} />
      </View>
    );
  }

  const busy = telegram.phase === "starting" || telegram.phase === "approved";
  return (
    <Button
      title={busy ? t("login.telegramSigningIn") : t("login.telegramButton")}
      disabled={busy}
      busy={busy}
      onPress={() => void telegram.start()}
    />
  );
}
