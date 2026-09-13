import { Component, type ErrorInfo, type ReactNode } from "react";
import { Pressable, Text, View } from "react-native";

export interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: (error: Error, reset: () => void) => ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
  message?: string;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, State> {
  override state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  override componentDidCatch(error: Error, info: ErrorInfo) {
    this.props.onError?.(error, info);
  }

  reset = () => this.setState({ error: null });

  override render() {
    const { error } = this.state;
    if (!error) return this.props.children;

    if (this.props.fallback) return this.props.fallback(error, this.reset);

    return (
      <ErrorState
        message={this.props.message ?? "The app hit an unexpected problem."}
        onRetry={this.reset}
      />
    );
  }
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <View
      style={{
        flex: 1,
        alignItems: "center",
        justifyContent: "center",
        gap: 14,
        padding: 24,
        backgroundColor: "#1a1a1a",
      }}
    >
      <Text style={{ color: "#f2f2f2", fontSize: 15, textAlign: "center", lineHeight: 22 }}>
        {message}
      </Text>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Try again"
        onPress={onRetry}
        style={{ borderWidth: 1, borderColor: "#6c6c6c", borderRadius: 8, paddingHorizontal: 18, paddingVertical: 10 }}
      >
        <Text style={{ color: "#f2f2f2", fontSize: 14, fontWeight: "600" }}>Try again</Text>
      </Pressable>
    </View>
  );
}
