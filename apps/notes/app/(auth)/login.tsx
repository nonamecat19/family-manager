import { toDisplayError, useClients, useTelegramLogin } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import Constants from "expo-constants";
import * as Linking from "expo-linking";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Text, View } from "react-native";

import { Body, Button, Caption, Display, Field, Screen, TextLink } from "@fm/ui";

import { strings } from "../../components/i18n/index.ts";

const telegramBot = (Constants.expoConfig?.extra as { telegramBot?: string } | undefined)?.telegramBot;

export default function LoginScreen() {
  const { auth } = useClients();
  const { signIn } = useAuth();
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
      if (mode === "register") await auth.register({ email, password, name });
      const res = await auth.login({ email, password });
      await signIn(tokensFromResponse(res, Date.now()));
    } catch (e) {
      const shown = toDisplayError(
        e,
        mode === "register" ? strings.login.registerError : strings.login.loginError,
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
        className="flex-1 justify-center gap-4.5 px-xl"
      >
        <View>
          <Display size={36}>{strings.app.name}</Display>
          <View className="mt-2.5">
            <Body>{mode === "login" ? strings.login.signInBody : strings.login.registerBody}</Body>
          </View>
        </View>

        {mode === "register" ? (
          <Field label={strings.login.name} value={name} onChangeText={setName} autoComplete="name" />
        ) : null}

        <Field
          label={strings.login.email}
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
        />
        <Field
          label={strings.login.password}
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          error={error ?? undefined}
        />

        {errorRef ? <Caption>{strings.common.errorReference(errorRef)}</Caption> : null}

        <Button
          title={mode === "login" ? strings.login.signIn : strings.login.createAccount}
          disabled={!canSubmit}
          busy={busy}
          onPress={() => void submit()}
        />

        <TextLink
          label={mode === "login" ? strings.login.switchToRegister : strings.login.switchToLogin}
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
  if (telegram.phase === "pending" && telegram.userCode) {
    return (
      <View className="items-center gap-2.5">
        <Text className="font-cap text-[22px] text-fg" style={{ letterSpacing: 3 }}>
          {telegram.userCode}
        </Text>
        <Caption>{strings.login.telegramPendingHint}</Caption>
        <TextLink label={strings.login.telegramCancel} onPress={telegram.cancel} />
      </View>
    );
  }

  const failure =
    telegram.phase === "denied"
      ? strings.login.telegramDenied
      : telegram.phase === "expired"
        ? strings.login.telegramExpired
        : telegram.phase === "error"
          ? toDisplayError(telegram.error, strings.login.telegramError).message
          : null;

  if (failure) {
    return (
      <View className="items-center gap-2.5">
        <Caption>{failure}</Caption>
        <Button title={strings.login.telegramRetry} onPress={() => void telegram.start()} />
      </View>
    );
  }

  const busy = telegram.phase === "starting" || telegram.phase === "approved";
  return (
    <Button
      title={busy ? strings.login.telegramSigningIn : strings.login.telegramButton}
      disabled={busy}
      busy={busy}
      onPress={() => void telegram.start()}
    />
  );
}

