import 'package:dio/dio.dart';
import 'package:finance_tracker/core/constants/api_constants.dart';
import 'package:finance_tracker/core/network/auth_interceptor.dart';
import 'package:finance_tracker/core/storage/secure_storage.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Creates and configures the [Dio] HTTP client.
///
/// Adds logging interceptor in debug mode and [AuthInterceptor]
/// for automatic token injection and refresh.
Dio createApiClient(SecureStorageService storage) {
  final dio = Dio(
    BaseOptions(
      baseUrl: ApiConstants.baseUrl,
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 10),
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    ),
  );

  if (kDebugMode) {
    dio.interceptors.add(
      LogInterceptor(
        requestBody: true,
        responseBody: true,
      ),
    );
  }

  // Separate Dio for refresh calls to avoid recursive interceptor invocation.
  final refreshDio = Dio(
    BaseOptions(
      baseUrl: ApiConstants.baseUrl,
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 10),
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    ),
  );

  dio.interceptors.add(AuthInterceptor(storage, refreshDio));

  return dio;
}

/// Provides the configured [Dio] HTTP client with auth interceptor.
final dioProvider = Provider<Dio>((ref) {
  final storage = ref.read(secureStorageProvider);
  return createApiClient(storage);
});
