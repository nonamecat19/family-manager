import {
  ActivityIndicator,
  Pressable,
  Text,
  TextInput,
  View,
  type PressableProps,
  type TextInputProps,
  type ViewProps,
} from "react-native";
import type { ReactNode } from "react";

/** A card surface. Every list row and panel in the apps sits on one of these. */
export function Card({ className = "", ...props }: ViewProps & { className?: string }) {
  return (
    <View
      className={`rounded-lg border border-border bg-surface dark:border-border-dark dark:bg-surface-dark ${className}`}
      {...props}
    />
  );
}

export interface ButtonProps extends Omit<PressableProps, "children"> {
  title: string;
  variant?: "primary" | "secondary" | "danger";
  loading?: boolean;
  className?: string;
}

export function Button({
  title,
  variant = "primary",
  loading = false,
  disabled,
  className = "",
  ...props
}: ButtonProps) {
  const tone =
    variant === "primary"
      ? "bg-primary"
      : variant === "danger"
        ? "bg-expense"
        : "bg-transparent border border-border dark:border-border-dark";
  const label = variant === "secondary" ? "text-fg dark:text-fg-dark" : "text-primary-fg";
  const isDisabled = disabled === true || loading;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: isDisabled, busy: loading }}
      disabled={isDisabled}
      className={`h-12 flex-row items-center justify-center rounded-md px-lg ${tone} ${
        isDisabled ? "opacity-50" : ""
      } ${className}`}
      {...props}
    >
      {loading ? (
        <ActivityIndicator color="#FFFFFF" />
      ) : (
        <Text className={`text-body font-semibold ${label}`}>{title}</Text>
      )}
    </Pressable>
  );
}

export interface FieldProps extends TextInputProps {
  label: string;
  error?: string;
}

export function Field({ label, error, className = "", ...props }: FieldProps) {
  return (
    <View className="gap-xs">
      <Text className="text-caption text-muted dark:text-muted-dark">{label}</Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor="#A0AEC0"
        className={`h-12 rounded-md border px-md text-body text-fg dark:text-fg-dark ${
          error ? "border-expense" : "border-border dark:border-border-dark"
        } ${className}`}
        {...props}
      />
      {error ? <Text className="text-caption text-expense">{error}</Text> : null}
    </View>
  );
}

/** Full-screen states, so no screen invents its own spinner or empty copy. */
export function Loading({ label = "Loading…" }: { label?: string }) {
  return (
    <View className="flex-1 items-center justify-center gap-md bg-bg dark:bg-bg-dark">
      <ActivityIndicator />
      <Text className="text-body text-muted dark:text-muted-dark">{label}</Text>
    </View>
  );
}

export function EmptyState({
  title,
  hint,
  action,
}: {
  title: string;
  hint?: string;
  action?: ReactNode;
}) {
  return (
    <View className="flex-1 items-center justify-center gap-sm p-xl">
      <Text className="text-title font-semibold text-fg dark:text-fg-dark">{title}</Text>
      {hint ? (
        <Text className="text-center text-body text-muted dark:text-muted-dark">{hint}</Text>
      ) : null}
      {action}
    </View>
  );
}

export function ErrorState({
  message,
  reference,
  retryable,
  onRetry,
}: {
  message: string;
  /**
   * The reference from an opaque internal error. Shown quietly under the message: it means
   * nothing to the user on its own, and it is the only thing that lets someone find the
   * request in the service logs once they report it.
   */
  reference?: string;
  /**
   * Whether trying again could plausibly work. When it is explicitly false the retry button is
   * hidden: refetching a PermissionDenied or a NotFound produces the same answer, and a button
   * that cannot help is worse than none. Omitted means "offer it", which is the old behaviour.
   */
  retryable?: boolean;
  onRetry?: () => void;
}) {
  return (
    <View className="flex-1 items-center justify-center gap-md p-xl">
      <Text className="text-title font-semibold text-expense">Something went wrong</Text>
      <Text className="text-center text-body text-muted dark:text-muted-dark">{message}</Text>
      {reference ? (
        <Text className="text-caption text-muted dark:text-muted-dark">Reference {reference}</Text>
      ) : null}
      {onRetry && retryable !== false ? (
        <Button title="Try again" variant="secondary" onPress={onRetry} />
      ) : null}
    </View>
  );
}

/** A coloured dot used for categories and accounts in dense lists. */
export function Dot({ color, size = 12 }: { color: string; size?: number }) {
  return (
    <View
      accessibilityElementsHidden
      style={{ width: size, height: size, borderRadius: size / 2, backgroundColor: color }}
    />
  );
}
