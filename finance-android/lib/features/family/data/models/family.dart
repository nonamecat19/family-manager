/// Model representing a family group.
class Family {
  /// Creates a [Family].
  const Family({
    required this.id,
    required this.name,
    required this.adminUserId,
    required this.createdAt,
  });

  /// Parses a [Family] from a JSON map.
  factory Family.fromJson(Map<String, dynamic> json) {
    return Family(
      id: json['id'] as String,
      name: json['name'] as String,
      adminUserId: json['admin_user_id'] as String,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }

  /// Unique identifier (UUID from server).
  final String id;

  /// Display name of the family group.
  final String name;

  /// User ID of the family admin.
  final String adminUserId;

  /// When the family was created.
  final DateTime createdAt;
}
