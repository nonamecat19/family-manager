import 'package:dio/dio.dart';
import 'package:finance_tracker/core/storage/secure_storage.dart';

/// Dio interceptor that injects auth tokens and handles 401 refresh.
///
/// Uses `QueuedInterceptor` to serialize error handling, preventing
/// race conditions when multiple requests hit 401 simultaneously.
///
/// Takes a separate `Dio` instance for refresh calls to avoid
/// recursive interceptor invocation.
class AuthInterceptor extends QueuedInterceptor {
  /// Creates an [AuthInterceptor].
  ///
  /// [_storage] is used to read/write tokens.
  /// [_refreshDio] is a plain Dio instance (no auth interceptor)
  /// used exclusively for token refresh calls.
  AuthInterceptor(this._storage, this._refreshDio);

  final SecureStorageService _storage;
  final Dio _refreshDio;

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = await _storage.readAccessToken();
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    if (err.response?.statusCode != 401) {
      handler.next(err);
      return;
    }

    final refreshToken = await _storage.readRefreshToken();
    if (refreshToken == null) {
      handler.next(err);
      return;
    }

    try {
      final response = await _refreshDio.post<Map<String, dynamic>>(
        '/auth/refresh',
        data: {'refresh_token': refreshToken},
      );
      final data = response.data!;
      final newAccess = data['access_token'] as String;
      final newRefresh = data['refresh_token'] as String;

      // Preserve existing user info, only update tokens.
      final userId = await _storage.readUserId();
      final email = await _storage.readEmail();
      await _storage.writeTokens(
        accessToken: newAccess,
        refreshToken: newRefresh,
        userId: userId ?? '',
        email: email ?? '',
      );

      // Retry original request with new token.
      err.requestOptions.headers['Authorization'] = 'Bearer $newAccess';
      final retryResponse = await _refreshDio.fetch<dynamic>(
        err.requestOptions,
      );
      handler.resolve(retryResponse);
    } on Exception {
      handler.next(err);
    }
  }
}
