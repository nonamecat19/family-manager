import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('parseAmountToCents', () {
    test('parses "12.34" to 1234', () {
      expect(Expense.parseAmountToCents('12.34'), 1234);
    });

    test('parses "1234.56" to 123456', () {
      expect(Expense.parseAmountToCents('1234.56'), 123456);
    });

    test('strips commas: "1,234.56" to 123456', () {
      expect(Expense.parseAmountToCents('1,234.56'), 123456);
    });

    test('returns null for "0" (non-positive)', () {
      expect(Expense.parseAmountToCents('0'), isNull);
    });

    test('returns null for "abc" (invalid)', () {
      expect(Expense.parseAmountToCents('abc'), isNull);
    });

    test('returns null for "-5" (negative)', () {
      expect(Expense.parseAmountToCents('-5'), isNull);
    });

    test('rounds correctly: "1.1" to 110', () {
      expect(Expense.parseAmountToCents('1.1'), 110);
    });
  });

  group('formatCents', () {
    test('formats 123456 as "\$1,234.56"', () {
      expect(Expense.formatCents(123456), r'$1,234.56');
    });

    test('formats 100 as "\$1.00"', () {
      expect(Expense.formatCents(100), r'$1.00');
    });

    test('formats 1 as "\$0.01"', () {
      expect(Expense.formatCents(1), r'$0.01');
    });
  });

  group('Expense.fromJson', () {
    test('correctly parses API response', () {
      final json = {
        'id': 'abc-123',
        'category_id': 'cat-456',
        'amount_cents': 5099,
        'note': 'Lunch',
        'expense_date': '2026-03-15',
      };

      final expense = Expense.fromJson(json);

      expect(expense.id, 'abc-123');
      expect(expense.categoryId, 'cat-456');
      expect(expense.amountCents, 5099);
      expect(expense.note, 'Lunch');
      expect(expense.expenseDate, DateTime(2026, 3, 15));
    });

    test('handles null note as empty string', () {
      final json = {
        'id': 'abc-123',
        'category_id': 'cat-456',
        'amount_cents': 100,
        'note': null,
        'expense_date': '2026-01-01',
      };

      final expense = Expense.fromJson(json);
      expect(expense.note, '');
    });
  });
}
