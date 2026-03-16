/// Per-member spending total in a family summary.
class MemberTotal {
  /// Creates a [MemberTotal].
  const MemberTotal({
    required this.userId,
    required this.userEmail,
    required this.totalCents,
    required this.expenseCount,
  });

  /// Parses a [MemberTotal] from a JSON map.
  factory MemberTotal.fromJson(Map<String, dynamic> json) {
    return MemberTotal(
      userId: json['user_id'] as String,
      userEmail: json['user_email'] as String,
      totalCents: json['total_cents'] as int,
      expenseCount: json['expense_count'] as int,
    );
  }

  /// User UUID.
  final String userId;

  /// Email of the family member.
  final String userEmail;

  /// Total spending in integer cents.
  final int totalCents;

  /// Number of expenses logged.
  final int expenseCount;
}

/// Per-category spending total in a family summary.
class FamilyCategoryTotal {
  /// Creates a [FamilyCategoryTotal].
  const FamilyCategoryTotal({
    required this.categoryId,
    required this.categoryName,
    required this.categoryColor,
    required this.categoryIcon,
    required this.totalCents,
    required this.expenseCount,
  });

  /// Parses a [FamilyCategoryTotal] from a JSON map.
  factory FamilyCategoryTotal.fromJson(Map<String, dynamic> json) {
    return FamilyCategoryTotal(
      categoryId: json['category_id'] as String,
      categoryName: json['category_name'] as String,
      categoryColor: json['category_color'] as String,
      categoryIcon: json['category_icon'] as String,
      totalCents: json['total_cents'] as int,
      expenseCount: json['expense_count'] as int,
    );
  }

  /// Category UUID.
  final String categoryId;

  /// Display name of the category.
  final String categoryName;

  /// Hex color string (e.g. "#FF7043").
  final String categoryColor;

  /// Icon name string (e.g. "restaurant").
  final String categoryIcon;

  /// Total spending in integer cents.
  final int totalCents;

  /// Number of expenses in this category.
  final int expenseCount;
}

/// Aggregated family spending summary for a month.
///
/// Contains the grand total plus breakdowns by person and by category.
class FamilySummary {
  /// Creates a [FamilySummary].
  const FamilySummary({
    required this.totalCents,
    required this.byPerson,
    required this.byCategory,
  });

  /// Parses a [FamilySummary] from the summary API JSON response.
  factory FamilySummary.fromJson(Map<String, dynamic> json) {
    return FamilySummary(
      totalCents: json['total_cents'] as int,
      byPerson: (json['by_person'] as List<dynamic>)
          .cast<Map<String, dynamic>>()
          .map(MemberTotal.fromJson)
          .toList(),
      byCategory: (json['by_category'] as List<dynamic>)
          .cast<Map<String, dynamic>>()
          .map(FamilyCategoryTotal.fromJson)
          .toList(),
    );
  }

  /// Grand total spending in integer cents.
  final int totalCents;

  /// Spending breakdown by family member.
  final List<MemberTotal> byPerson;

  /// Spending breakdown by category.
  final List<FamilyCategoryTotal> byCategory;
}
