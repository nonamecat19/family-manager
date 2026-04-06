/// Model representing a member of a family group.
class FamilyMember {
  /// Creates a [FamilyMember].
  const FamilyMember({
    required this.id,
    required this.userId,
    required this.email,
    required this.role,
    required this.joinedAt,
  });

  /// Parses a [FamilyMember] from a JSON map.
  factory FamilyMember.fromJson(Map<String, dynamic> json) {
    return FamilyMember(
      id: json['id'] as String,
      userId: json['user_id'] as String,
      email: json['email'] as String,
      role: json['role'] as String,
      joinedAt: DateTime.parse(json['joined_at'] as String),
    );
  }

  /// Unique identifier for the membership record.
  final String id;

  /// The user's ID.
  final String userId;

  /// The user's email address.
  final String email;

  /// Role within the family ("admin" or "member").
  final String role;

  /// When the user joined the family.
  final DateTime joinedAt;

  /// Whether this member has admin privileges.
  bool get isAdmin => role == 'admin';
}
