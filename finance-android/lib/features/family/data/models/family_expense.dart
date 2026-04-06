/// A single expense from the family expense feed.
///
/// Includes the spender's email so the UI can display who spent what.
class FamilyExpense {
  /// Creates a [FamilyExpense].
  const FamilyExpense({
    required this.id,
    required this.userId,
    required this.userEmail,
    required this.categoryId,
    required this.categoryName,
    required this.categoryColor,
    required this.categoryIcon,
    required this.amountCents,
    required this.note,
    required this.expenseDate,
  });

  /// Parses a [FamilyExpense] from a JSON map (API response).
  factory FamilyExpense.fromJson(Map<String, dynamic> json) {
    return FamilyExpense(
      id: json['id'] as String,
      userId: json['user_id'] as String,
      userEmail: json['user_email'] as String,
      categoryId: json['category_id'] as String,
      categoryName: json['category_name'] as String,
      categoryColor: json['category_color'] as String,
      categoryIcon: json['category_icon'] as String,
      amountCents: json['amount_cents'] as int,
      note: (json['note'] as String?) ?? '',
      expenseDate: DateTime.parse(json['expense_date'] as String),
    );
  }

  /// Expense UUID.
  final String id;

  /// User UUID of the spender.
  final String userId;

  /// Email of the user who made this expense.
  final String userEmail;

  /// Category UUID.
  final String categoryId;

  /// Display name of the category.
  final String categoryName;

  /// Hex color string (e.g. "#FF7043").
  final String categoryColor;

  /// Icon name string (e.g. "restaurant").
  final String categoryIcon;

  /// Amount in integer cents (e.g. 1234 = $12.34).
  final int amountCents;

  /// Optional user note.
  final String note;

  /// The date the expense occurred.
  final DateTime expenseDate;
}
