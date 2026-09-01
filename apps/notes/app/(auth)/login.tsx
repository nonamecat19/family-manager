import { toDisplayError, useClients } from "@fm/api";
import { tokensFromResponse, useAuth } from "@fm/auth";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, Text, TextInput, View } from "react-native";

import { strings } from "../../components/i18n/index.ts";
import { PrimaryButton, Screen, nocturne } from "../../components/nocturne/index.ts";

/** Sign in / register. The recipes screen's flow, drawn in Nocturne. */
export default function LoginScreen() {
  const { auth } = useClients();
  const { signIn } = useAuth();

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
      // The service phrases the useful failures itself ("that email is already registered");
      // toDisplayError keeps those and falls back to the generic line only when there is
      // nothing readable in the error.
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
        className="flex-1 justify-center gap-[18px] px-[24px]"
      >
        <View className="gap-[10px]">
          <View className="flex-row items-center gap-[10px]">
            <View className="h-[26px] w-[26px] items-center justify-center rounded-md border border-accent">
              <Text className="font-semi text-[13px] text-accent">C</Text>
            </View>
            <Text className="font-semi text-[30px] text-fg" style={{ letterSpacing: -0.6 }}>
              {strings.app.name}
            </Text>
          </View>
          <Text className="font-sans text-[15px] leading-[23px] text-neutral-400">
            {mode === "login" ? strings.login.signInBody : strings.login.registerBody}
          </Text>
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

        {errorRef ? (
          <Text className="font-sans text-[13px] text-neutral-600">
            {strings.common.errorReference(errorRef)}
          </Text>
        ) : null}

        <PrimaryButton
          title={
            busy
              ? strings.common.oneMoment
              : mode === "login"
                ? strings.login.signIn
                : strings.login.createAccount
          }
          disabled={!canSubmit}
          onPress={() => void submit()}
        />

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={
            mode === "login" ? strings.login.switchToRegister : strings.login.switchToLogin
          }
          onPress={() => {
            setMode(mode === "login" ? "register" : "login");
            setError(null);
          }}
        >
          <Text className="text-center font-med text-[14px] text-accent">
            {mode === "login" ? strings.login.switchToRegister : strings.login.switchToLogin}
          </Text>
        </Pressable>
      </KeyboardAvoidingView>
    </Screen>
  );
}

type FieldProps = React.ComponentProps<typeof TextInput> & { label: string; error?: string };

/**
 * @fm/ui's Field is drawn for the shared blue-grey system; Nocturne's input is a dark surface
 * with a token border. Local rather than a change to the shared primitive, which four apps
 * render.
 */
function Field({ label, error, ...props }: FieldProps) {
  return (
    <View className="gap-[6px]">
      <Text className="font-med text-[12px] text-neutral-500">{label}</Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={nocturne.neutral[600]}
        {...props}
        className={`h-[46px] rounded-md border bg-surface px-[14px] font-sans text-[15px] text-fg ${
          error ? "border-error" : "border-neutral-800"
        }`}
      />
      {error ? <Text className="font-sans text-[12.5px] text-error">{error}</Text> : null}
    </View>
  );
}
