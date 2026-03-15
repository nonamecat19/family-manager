/// Response model for authentication API calls (login/signup).
class AuthResponse {
  /// Creates an [AuthResponse].
  const AuthResponse({
    required this.accessToken,
    required this.refreshToken,
    required this.userId,
    required this.email,
  });

  /// Creates an [AuthResponse] from a JSON map matching the backend shape.
  ///
  /// Expected shape:
  /// ```json
  /// {
  ///   "access_token": "...",
  ///   "refresh_token": "...",
  ///   "user": { "id": "...", "email": "..." }
  /// }
  /// ```
  factory AuthResponse.fromJson(Map<String, dynamic> json) {
    final user = json['user'] as Map<String, dynamic>;
    return AuthResponse(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
      userId: user['id'] as String,
      email: user['email'] as String,
    );
  }

  /// The JWT access token.
  final String accessToken;

  /// The JWT refresh token.
  final String refreshToken;

  /// The authenticated user's ID.
  final String userId;

  /// The authenticated user's email.
  final String email;
}
