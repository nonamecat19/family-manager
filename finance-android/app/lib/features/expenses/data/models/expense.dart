import 'package:intl/intl.dart';

/// Model representing a user expense.
class Expense {
  /// Creates an [Expense].
  const Expense({
    required this.id,
    required this.categoryId,
    required this.amountCents,
    required this.note,
    required this.expenseDate,
  });

  /// Parses an [Expense] from a JSON map (API response).
  factory Expense.fromJson(Map<String, dynamic> json) {
    return Expense(
      id: json['id'] as String,
      categoryId: json['category_id'] as String,
      amountCents: json['amount_cents'] as int,
      note: (json['note'] as String?) ?? '',
      expenseDate: DateTime.parse(json['expense_date'] as String),
    );
  }

  /// Unique identifier (UUID from server).
  final String id;

  /// The category this expense belongs to.
  final String categoryId;

  /// Amount in integer cents (e.g. 1234 = $12.34).
  final int amountCents;

  /// Optional user note.
  final String note;

  /// The date the expense occurred.
  final DateTime expenseDate;

  /// Parses a user-entered amount string to integer cents.
  ///
  /// Returns `null` if the input is invalid or non-positive.
  /// Strips commas, parses as double, multiplies by 100 and rounds.
  static int? parseAmountToCents(String input) {
    final cleaned = input.replaceAll(',', '');
    final value = double.tryParse(cleaned);
    if (value == null || value <= 0) return null;
    return (value * 100).round();
  }

  /// Formats integer cents as a currency string (e.g. 1234 -> "$12.34").
  static String formatCents(int cents) {
    return NumberFormat.currency(locale: 'en_US', symbol: r'$')
        .format(cents / 100);
  }
}
