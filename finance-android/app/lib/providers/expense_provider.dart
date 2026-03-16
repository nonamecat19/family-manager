import 'package:finance_tracker/features/expenses/data/expense_repository.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Manages expense state for the application.
///
/// Handles creating expenses and loading the expense list
/// via [ExpenseRepository].
class ExpenseNotifier extends StateNotifier<ExpenseState> {
  /// Creates an [ExpenseNotifier].
  ExpenseNotifier(this._repository) : super(const ExpenseInitial());

  final ExpenseRepository _repository;

  /// Creates a new expense and prepends it to the loaded list.
  ///
  /// Returns `true` on success, `false` on failure.
  Future<bool> createExpense({
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async {
    try {
      final expense = await _repository.createExpense(
        categoryId: categoryId,
        amountCents: amountCents,
        note: note,
        expenseDate: expenseDate,
      );

      // Prepend new expense so it appears at the top immediately.
      final currentExpenses = switch (state) {
        ExpenseLoaded(:final expenses) => expenses,
        _ => <Expense>[],
      };
      state = ExpenseLoaded([expense, ...currentExpenses]);
      return true;
    } on Exception catch (e) {
      state = ExpenseError(e.toString());
      return false;
    }
  }

  /// Loads expenses from the API with optional filters.
  Future<void> loadExpenses({
    int limit = 50,
    int offset = 0,
    String? dateFrom,
    String? dateTo,
    String? categoryId,
  }) async {
    state = const ExpenseLoading();
    try {
      final expenses = await _repository.getExpenses(
        limit: limit,
        offset: offset,
        dateFrom: dateFrom,
        dateTo: dateTo,
        categoryId: categoryId,
      );
      state = ExpenseLoaded(expenses);
    } on Exception catch (e) {
      state = ExpenseError(e.toString());
    }
  }

  /// Updates an expense and replaces it in the local list.
  ///
  /// Returns `true` on success, `false` on failure.
  Future<bool> updateExpense({
    required String id,
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async {
    try {
      final updated = await _repository.updateExpense(
        id: id,
        categoryId: categoryId,
        amountCents: amountCents,
        note: note,
        expenseDate: expenseDate,
      );
      final currentExpenses = switch (state) {
        ExpenseLoaded(:final expenses) => expenses,
        _ => <Expense>[],
      };
      state = ExpenseLoaded([
        for (final e in currentExpenses)
          if (e.id == id) updated else e,
      ]);
      return true;
    } on Exception catch (e) {
      state = ExpenseError(e.toString());
      return false;
    }
  }

  /// Deletes an expense and removes it from the local list.
  ///
  /// Returns `true` on success, `false` on failure.
  Future<bool> deleteExpense(String id) async {
    try {
      await _repository.deleteExpense(id);
      final currentExpenses = switch (state) {
        ExpenseLoaded(:final expenses) => expenses,
        _ => <Expense>[],
      };
      state = ExpenseLoaded(
        currentExpenses.where((e) => e.id != id).toList(),
      );
      return true;
    } on Exception catch (e) {
      state = ExpenseError(e.toString());
      return false;
    }
  }
}

/// Provides the expense state and notifier.
final expenseStateProvider =
    StateNotifierProvider<ExpenseNotifier, ExpenseState>((ref) {
  return ExpenseNotifier(ref.read(expenseRepositoryProvider));
});
