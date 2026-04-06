import 'package:dio/dio.dart';
import 'package:finance_tracker/features/auth/data/models/auth_response.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Repository for authentication API calls.
///
/// Communicates with the Go backend auth endpoints
/// at `/api/v1/auth/*`.
class AuthRepository {
  /// Creates an [AuthRepository] with the given [Dio] client.
  const AuthRepository(this._dio);

  final Dio _dio;

  /// Registers a new user with [email] and [password].
  ///
  /// Returns an [AuthResponse] on success.
  /// Throws [AuthException] on failure with parsed server errors.
  Future<AuthResponse> signup(String email, String password) async {
    try {
      final response = await _dio.post<Map<String, dynamic>>(
        '/auth/signup',
        data: {'email': email, 'password': password},
      );
      return AuthResponse.fromJson(response.data!);
    } on DioException catch (e) {
      throw _parseError(e);
    }
  }

  /// Logs in a user with [email] and [password].
  ///
  /// Returns an [AuthResponse] on success.
  /// Throws [AuthException] on failure with parsed server errors.
  Future<AuthResponse> login(String email, String password) async {
    try {
      final response = await _dio.post<Map<String, dynamic>>(
        '/auth/login',
        data: {'email': email, 'password': password},
      );
      return AuthResponse.fromJson(response.data!);
    } on DioException catch (e) {
      throw _parseError(e);
    }
  }

  /// Refreshes the token pair using [refreshToken].
  ///
  /// Returns a map with `access_token` and `refresh_token` on success.
  /// Throws [AuthException] on failure.
  Future<Map<String, String>> refresh(String refreshToken) async {
    try {
      final response = await _dio.post<Map<String, dynamic>>(
        '/auth/refresh',
        data: {'refresh_token': refreshToken},
      );
      return {
        'access_token': response.data!['access_token'] as String,
        'refresh_token': response.data!['refresh_token'] as String,
      };
    } on DioException catch (e) {
      throw _parseError(e);
    }
  }

  /// Logs out by revoking the [refreshToken].
  ///
  /// Throws [AuthException] on failure.
  Future<void> logout(String refreshToken) async {
    try {
      await _dio.post<void>(
        '/auth/logout',
        data: {'refresh_token': refreshToken},
      );
    } on DioException catch (e) {
      throw _parseError(e);
    }
  }

  AuthException _parseError(DioException e) {
    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final error = data['error'] as String? ?? 'An unexpected error occurred';

      // Map server error messages to field-specific errors.
      final fieldErrors = <String, String>{};
      if (error.toLowerCase().contains('no account with this email') ||
          error.toLowerCase().contains('user not found')) {
        fieldErrors['email'] = 'No account with this email';
      } else if (error.toLowerCase().contains('wrong password') ||
          error.toLowerCase().contains('invalid password')) {
        fieldErrors['password'] = 'Wrong password';
      } else if (error.toLowerCase().contains('already exists') ||
          error.toLowerCase().contains('duplicate')) {
        fieldErrors['email'] = 'An account with this email already exists';
      } else if (error.toLowerCase().contains('email')) {
        fieldErrors['email'] = error;
      } else if (error.toLowerCase().contains('password')) {
        fieldErrors['password'] = error;
      }

      return AuthException(
        error,
        fieldErrors: fieldErrors.isEmpty ? null : fieldErrors,
      );
    }
    return AuthException(
      e.message ?? 'An unexpected error occurred',
    );
  }
}

/// Exception thrown by [AuthRepository] with parsed error info.
class AuthException implements Exception {
  /// Creates an [AuthException].
  const AuthException(this.message, {this.fieldErrors});

  /// General error message from the server.
  final String message;

  /// Optional field-specific errors for inline display.
  final Map<String, String>? fieldErrors;

  @override
  String toString() => 'AuthException: $message';
}

/// Provides an [AuthRepository] using the app's Dio client.
final authRepositoryProvider = Provider<AuthRepository>((ref) {
  // Use a plain Dio for auth calls (no auth interceptor to avoid recursion).
  final dio = Dio(
    BaseOptions(
      baseUrl: 'http://localhost:8080/api/v1',
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 10),
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    ),
  );
  return AuthRepository(dio);
});
