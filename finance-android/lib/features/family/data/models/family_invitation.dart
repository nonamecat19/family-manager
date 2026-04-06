/// Model representing a pending family invitation.
class FamilyInvitation {
  /// Creates a [FamilyInvitation].
  const FamilyInvitation({
    required this.id,
    required this.expiresAt,
    required this.createdAt,
  });

  /// Parses a [FamilyInvitation] from a JSON map.
  factory FamilyInvitation.fromJson(Map<String, dynamic> json) {
    return FamilyInvitation(
      id: json['id'] as String,
      expiresAt: DateTime.parse(json['expires_at'] as String),
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }

  /// Unique identifier for the invitation.
  final String id;

  /// When the invitation expires.
  final DateTime expiresAt;

  /// When the invitation was created.
  final DateTime createdAt;
}
