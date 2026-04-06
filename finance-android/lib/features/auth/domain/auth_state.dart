/// Represents the authentication state of the application.
sealed class AuthState {
  const AuthState();
}

/// Initial state before session restoration attempt.
class AuthInitial extends AuthState {
  const AuthInitial();
}

/// Loading state during login, signup, or session restore.
class AuthLoading extends AuthState {
  const AuthLoading();
}

/// User is authenticated with valid tokens.
class Authenticated extends AuthState {
  /// Creates an [Authenticated] state.
  const Authenticated({required this.userId, required this.email});

  /// The authenticated user's ID.
  final String userId;

  /// The authenticated user's email.
  final String email;
}

/// User is not authenticated (logged out or session expired).
class Unauthenticated extends AuthState {
  const Unauthenticated();
}

/// Authentication failed with an error.
class AuthError extends AuthState {
  /// Creates an [AuthError] state.
  const AuthError(this.message, {this.fieldErrors});

  /// General error message.
  final String message;

  /// Optional field-specific errors for inline display.
  ///
  /// Keys are field names (e.g., "email", "password").
  /// Values are the error messages to display under each field.
  final Map<String, String>? fieldErrors;
}
