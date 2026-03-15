import 'package:finance_tracker/core/storage/secure_storage.dart';
import 'package:finance_tracker/features/auth/data/auth_repository.dart';
import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Manages authentication state for the application.
///
/// Handles login, signup, logout, and session restoration
/// via [AuthRepository] and [SecureStorageService].
class AuthNotifier extends StateNotifier<AuthState> {
  /// Creates an [AuthNotifier].
  AuthNotifier(this._repository, this._storage) : super(const AuthInitial());

  final AuthRepository _repository;
  final SecureStorageService _storage;

  /// Logs in with [email] and [password].
  ///
  /// Sets [Authenticated] on success, [AuthError] on failure.
  Future<void> login(String email, String password) async {
    state = const AuthLoading();
    try {
      final response = await _repository.login(email, password);
      await _storage.writeTokens(
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
        userId: response.userId,
        email: response.email,
      );
      state = Authenticated(userId: response.userId, email: response.email);
    } on AuthException catch (e) {
      state = AuthError(e.message, fieldErrors: e.fieldErrors);
    } on Exception catch (e) {
      state = AuthError(e.toString());
    }
  }

  /// Creates an account with [email] and [password].
  ///
  /// Sets [Authenticated] on success, [AuthError] on failure.
  Future<void> signup(String email, String password) async {
    state = const AuthLoading();
    try {
      final response = await _repository.signup(email, password);
      await _storage.writeTokens(
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
        userId: response.userId,
        email: response.email,
      );
      state = Authenticated(userId: response.userId, email: response.email);
    } on AuthException catch (e) {
      state = AuthError(e.message, fieldErrors: e.fieldErrors);
    } on Exception catch (e) {
      state = AuthError(e.toString());
    }
  }

  /// Logs out the current user.
  ///
  /// Revokes the refresh token on the server and clears local storage.
  Future<void> logout() async {
    final refreshToken = await _storage.readRefreshToken();
    if (refreshToken != null) {
      try {
        await _repository.logout(refreshToken);
      } on Exception {
        // Best-effort server logout; clear local state regardless.
      }
    }
    await _storage.clearTokens();
    state = const Unauthenticated();
  }

  /// Attempts to restore a previous session on app startup.
  ///
  /// Reads tokens from secure storage and tries a refresh.
  /// Sets [Authenticated] if refresh succeeds, [Unauthenticated] otherwise.
  Future<void> tryRestoreSession() async {
    state = const AuthLoading();
    final refreshToken = await _storage.readRefreshToken();
    if (refreshToken == null) {
      state = const Unauthenticated();
      return;
    }

    try {
      final tokens = await _repository.refresh(refreshToken);
      final userId = await _storage.readUserId();
      final email = await _storage.readEmail();

      if (userId == null || email == null) {
        await _storage.clearTokens();
        state = const Unauthenticated();
        return;
      }

      await _storage.writeTokens(
        accessToken: tokens['access_token']!,
        refreshToken: tokens['refresh_token']!,
        userId: userId,
        email: email,
      );
      state = Authenticated(userId: userId, email: email);
    } on Exception {
      await _storage.clearTokens();
      state = const Unauthenticated();
    }
  }
}

/// Provides the authentication state and notifier.
final authStateProvider =
    StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(
    ref.read(authRepositoryProvider),
    ref.read(secureStorageProvider),
  );
});
