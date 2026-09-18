import { toDisplayError, useClients, useTelegramLogin } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import Constants from "expo-constants";
import * as Linking from "expo-linking";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Text, View } from "react-native";

import { Body, Button, Caption, Display, Field, Screen, TextLink, Icon, organic } from "@fm/ui";

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
        className="flex-1 justify-center gap-4.5 px-xl"
      >
        <View>
          <View className="mb-lg h-[44px] w-[44px] items-center justify-center rounded-full border border-accent">
            <Icon name="wallet" size={22} color={organic.accent[600]} />
          </View>
          <Display size={36}>{t("auth.title")}</Display>
          <View className="mt-2.5">
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

        {telegram.available ? <TelegramLogin telegram={telegram} /> : null}
      </KeyboardAvoidingView>
    </Screen>
  );
}

function TelegramLogin({ telegram }: { telegram: ReturnType<typeof useTelegramLogin> }) {
  const { t } = useI18n();

  if (telegram.phase === "pending" && telegram.userCode) {
    return (
      <View className="items-center gap-2.5">
        <Text className="text-[22px] font-fig-semi text-fg" style={{ letterSpacing: 3 }}>
          {telegram.userCode}
        </Text>
        <Caption>{t("auth.telegramPendingHint")}</Caption>
        <TextLink label={t("auth.telegramCancel")} onPress={telegram.cancel} />
      </View>
    );
  }

  const failure =
    telegram.phase === "denied"
      ? t("auth.telegramDenied")
      : telegram.phase === "expired"
        ? t("auth.telegramExpired")
        : telegram.phase === "error"
          ? toDisplayError(telegram.error, t("auth.telegramError")).message
          : null;

  if (failure) {
    return (
      <View className="items-center gap-2.5">
        <Caption>{failure}</Caption>
        <Button title={t("auth.telegramRetry")} onPress={() => void telegram.start()} />
      </View>
    );
  }

  const busy = telegram.phase === "starting" || telegram.phase === "approved";
  return (
    <Button
      title={busy ? t("auth.telegramSigningIn") : t("auth.telegramButton")}
      disabled={busy}
      busy={busy}
      onPress={() => void telegram.start()}
    />
  );
}
