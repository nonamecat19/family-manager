import 'package:dio/dio.dart';
import 'package:finance_tracker/core/network/api_client.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Repository for expense API calls.
///
/// Communicates with the Go backend expense endpoints
/// at `/expenses`.
class ExpenseRepository {
  /// Creates an [ExpenseRepository] with the given [Dio] client.
  const ExpenseRepository(this._dio);

  final Dio _dio;

  /// Creates a new expense.
  ///
  /// [amountCents] is the amount in integer cents.
  /// [expenseDate] is formatted as "2006-01-02".
  Future<Expense> createExpense({
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/expenses',
      data: {
        'category_id': categoryId,
        'amount_cents': amountCents,
        'note': note,
        'expense_date': expenseDate,
      },
    );
    return Expense.fromJson(response.data!);
  }

  /// Fetches expenses for the authenticated user with pagination.
  Future<List<Expense>> getExpenses({int limit = 50, int offset = 0}) async {
    final response = await _dio.get<List<dynamic>>(
      '/expenses',
      queryParameters: {'limit': limit, 'offset': offset},
    );
    return response.data!
        .cast<Map<String, dynamic>>()
        .map(Expense.fromJson)
        .toList();
  }
}

/// Provides an [ExpenseRepository] using the app's Dio client.
final expenseRepositoryProvider = Provider<ExpenseRepository>((ref) {
  return ExpenseRepository(ref.read(dioProvider));
});
