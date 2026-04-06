import 'package:dio/dio.dart';
import 'package:finance_tracker/core/network/auth_interceptor.dart';
import 'package:finance_tracker/core/storage/secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

class MockSecureStorageService extends Mock implements SecureStorageService {}

void main() {
  late MockSecureStorageService mockStorage;
  late Dio dio;
  late Dio refreshDio;

  setUp(() {
    mockStorage = MockSecureStorageService();
    refreshDio = Dio(BaseOptions(baseUrl: 'http://localhost'));
    dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
    dio.interceptors.add(AuthInterceptor(mockStorage, refreshDio));
  });

  group('AuthInterceptor', () {
    test('adds Authorization header when access token exists', () async {
      when(() => mockStorage.readAccessToken())
          .thenAnswer((_) async => 'test-access-token');

      // Use a second interceptor to capture the request after AuthInterceptor
      // has modified it, then reject to avoid actual HTTP call.
      RequestOptions? capturedOptions;
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            capturedOptions = options;
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.cancel,
              ),
            );
          },
        ),
      );

      try {
        await dio.get<void>('/test');
      } on DioException {
        // Expected: we rejected the request.
      }

      verify(() => mockStorage.readAccessToken()).called(1);
      expect(capturedOptions, isNotNull);
      expect(
        capturedOptions!.headers['Authorization'],
        equals('Bearer test-access-token'),
      );
    });

    test('passes through without header when no token', () async {
      when(() => mockStorage.readAccessToken())
          .thenAnswer((_) async => null);

      RequestOptions? capturedOptions;
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            capturedOptions = options;
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.cancel,
              ),
            );
          },
        ),
      );

      try {
        await dio.get<void>('/test');
      } on DioException {
        // Expected: we rejected the request.
      }

      verify(() => mockStorage.readAccessToken()).called(1);
      expect(capturedOptions, isNotNull);
      expect(capturedOptions!.headers['Authorization'], isNull);
    });
  });
}
