import { Component, type ErrorInfo, type ReactNode } from "react";

import { ErrorState } from "./primitives.tsx";

export interface ErrorBoundaryProps {
  children: ReactNode;
  /**
   * Rendered instead of the default screen. Receives the error and a function that clears the
   * boundary so the tree can try to mount again.
   */
  fallback?: (error: Error, reset: () => void) => ReactNode;
  /**
   * Called once per caught error, before anything is rendered. This is where a report goes;
   * the boundary deliberately does not know about any particular reporting mechanism.
   */
  onError?: (error: Error, info: ErrorInfo) => void;
  /** Message for the default screen. Supplied by the app so it can be translated. */
  message?: string;
}

interface State {
  error: Error | null;
}

/**
 * Catches a render-time crash and shows a screen instead of unmounting the tree.
 *
 * Without one, a thrown error during render unmounts everything above it. In development that
 * is the red box, which is why this is easy to go without; in a release build it is a blank
 * screen with no way back, and the user's only recourse is to force-quit the app.
 *
 * It catches render, lifecycle and constructor errors only — not rejected promises, not errors
 * thrown in an event handler, and not anything asynchronous. Those paths already return through
 * TanStack Query, which has its own error state.
 *
 * A class component because React has no hook equivalent: componentDidCatch has no counterpart
 * in the hooks API, and every "hook" version in circulation is a class in a wrapper.
 */
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
