import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Wrapper around [FlutterSecureStorage] for token persistence.
///
/// Provides typed access to access and refresh tokens stored
/// in the platform's secure key-value store.
class SecureStorageService {
  /// Creates a [SecureStorageService] with the given [FlutterSecureStorage].
  const SecureStorageService(this._storage);

  final FlutterSecureStorage _storage;

  static const _accessTokenKey = 'access_token';
  static const _refreshTokenKey = 'refresh_token';
  static const _userIdKey = 'user_id';
  static const _emailKey = 'user_email';

  /// Reads the stored access token, or `null` if none.
  Future<String?> readAccessToken() => _storage.read(key: _accessTokenKey);

  /// Reads the stored refresh token, or `null` if none.
  Future<String?> readRefreshToken() => _storage.read(key: _refreshTokenKey);

  /// Reads the stored user ID, or `null` if none.
  Future<String?> readUserId() => _storage.read(key: _userIdKey);

  /// Reads the stored user email, or `null` if none.
  Future<String?> readEmail() => _storage.read(key: _emailKey);

  /// Persists token pair and user info to secure storage.
  Future<void> writeTokens({
    required String accessToken,
    required String refreshToken,
    required String userId,
    required String email,
  }) async {
    await _storage.write(key: _accessTokenKey, value: accessToken);
    await _storage.write(key: _refreshTokenKey, value: refreshToken);
    await _storage.write(key: _userIdKey, value: userId);
    await _storage.write(key: _emailKey, value: email);
  }

  /// Clears all stored tokens and user info.
  Future<void> clearTokens() async {
    await _storage.delete(key: _accessTokenKey);
    await _storage.delete(key: _refreshTokenKey);
    await _storage.delete(key: _userIdKey);
    await _storage.delete(key: _emailKey);
  }
}

/// Provides the [SecureStorageService] singleton.
final secureStorageProvider = Provider<SecureStorageService>((ref) {
  return const SecureStorageService(FlutterSecureStorage());
});
