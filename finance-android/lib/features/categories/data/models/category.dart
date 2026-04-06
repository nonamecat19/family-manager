/// Model representing an expense category.
///
/// Categories have a visual identity (icon name + hex color string)
/// and a user-controlled sort order.
class Category {
  /// Creates a [Category].
  const Category({
    required this.id,
    required this.name,
    required this.icon,
    required this.color,
    required this.sortOrder,
  });

  /// Parses a [Category] from a JSON map.
  ///
  /// Expects `sort_order` (snake_case) from the API response.
  factory Category.fromJson(Map<String, dynamic> json) {
    return Category(
      id: json['id'] as String,
      name: json['name'] as String,
      icon: json['icon'] as String,
      color: json['color'] as String,
      sortOrder: json['sort_order'] as int,
    );
  }

  /// Unique identifier (UUID from server).
  final String id;

  /// Display name chosen by the user.
  final String name;

  /// Material Icon name string, e.g. "restaurant".
  final String icon;

  /// Hex color string, e.g. "#FF5722".
  final String color;

  /// User-controlled sort position (ascending).
  final int sortOrder;

  /// Serializes to JSON for create/update requests.
  ///
  /// Excludes [id] and [sortOrder] which are server-managed.
  Map<String, dynamic> toJson() => {
        'name': name,
        'icon': icon,
        'color': color,
      };

  /// Creates a copy with optional field overrides.
  Category copyWith({
    String? id,
    String? name,
    String? icon,
    String? color,
    int? sortOrder,
  }) {
    return Category(
      id: id ?? this.id,
      name: name ?? this.name,
      icon: icon ?? this.icon,
      color: color ?? this.color,
      sortOrder: sortOrder ?? this.sortOrder,
    );
  }
}
