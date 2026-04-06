import 'package:finance_tracker/core/database/local_expense_source.dart';
import 'package:finance_tracker/features/expenses/data/expense_repository.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:finance_tracker/features/sync/data/sync_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

/// Manages expense state for the application.
///
/// Handles creating expenses (local-first with background sync)
/// and loading the expense list via [ExpenseRepository].
class ExpenseNotifier extends StateNotifier<ExpenseState> {
  /// Creates an [ExpenseNotifier].
  ExpenseNotifier(this._repository, this._localSource, this._syncService)
      : super(const ExpenseInitial());

  final ExpenseRepository _repository;
  final LocalExpenseSource _localSource;
  final SyncService _syncService;

  /// Creates a new expense locally, then syncs in the background.
  ///
  /// Returns `true` on success (local write always succeeds).
  Future<bool> createExpense({
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async {
    final localId = const Uuid().v4();
    final expense = Expense(
      id: localId,
      categoryId: categoryId,
      amountCents: amountCents,
      note: note,
      expenseDate: DateTime.parse(expenseDate),
      synced: false,
    );

    // Write to local DB first (offline-safe).
    await _localSource.insertExpense(expense);

    // Prepend to UI state immediately.
    final currentExpenses = switch (state) {
      ExpenseLoaded(:final expenses) => expenses,
      _ => <Expense>[],
    };
    state = ExpenseLoaded([expense, ...currentExpenses]);

    // Attempt sync in background (non-blocking).
    _syncService.syncPendingExpenses().then((result) {
      if (result.syncedCount > 0) {
        // Mark synced expenses in UI state.
        final current = switch (state) {
          ExpenseLoaded(:final expenses) => expenses,
          _ => <Expense>[],
        };
        state = ExpenseLoaded([
          for (final e in current)
            if (e.id == localId) e.copyWith(synced: true) else e,
        ]);
      }
    });

    return true;
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
  return ExpenseNotifier(
    ref.read(expenseRepositoryProvider),
    ref.read(localExpenseSourceProvider),
    ref.read(syncServiceProvider),
  );
});
