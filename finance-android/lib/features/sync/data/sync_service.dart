import 'package:dio/dio.dart';
import 'package:finance_tracker/core/database/local_category_source.dart';
import 'package:finance_tracker/core/database/local_expense_source.dart';
import 'package:finance_tracker/features/expenses/data/expense_repository.dart';
import 'package:finance_tracker/features/sync/domain/sync_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Background sync engine that processes pending local expenses.
///
/// Reads unsynced expenses from [LocalExpenseSource], sends each to the
/// server via [ExpenseRepository], and marks them as synced on success.
/// Uses a mutex flag to prevent concurrent sync runs.
class SyncService {
  /// Creates a [SyncService].
  SyncService(this._localSource, this._repository);

  final LocalExpenseSource _localSource;
  final ExpenseRepository _repository;
  bool _syncing = false;

  /// Processes all pending (unsynced) expenses.
  ///
  /// Returns a [SyncResult] with the count of synced and failed expenses.
  /// Stops processing on the first [DioException] to avoid hammering
  /// a broken connection.
  Future<SyncResult> syncPendingExpenses() async {
    if (_syncing) return const SyncResult();
    _syncing = true;

    var syncedCount = 0;
    var failedCount = 0;

    try {
      final pending = await _localSource.getUnsyncedExpenses();

      for (final row in pending) {
        try {
          final serverExpense = await _repository.createExpense(
            categoryId: row['category_id'] as String,
            amountCents: row['amount_cents'] as int,
            note: row['note'] as String,
            expenseDate: row['expense_date'] as String,
          );
          await _localSource.markSynced(
            row['id'] as String,
            serverExpense.id,
          );
          syncedCount++;
        } on DioException {
          failedCount++;
          break;
        }
      }
    } finally {
      _syncing = false;
    }

    return SyncResult(syncedCount: syncedCount, failedCount: failedCount);
  }
}

/// Provides the local expense source.
///
/// Must be overridden after database initialization in main.dart.
final localExpenseSourceProvider = Provider<LocalExpenseSource>((ref) {
  throw UnimplementedError('Must be overridden after DB init');
});

/// Provides the local category source.
///
/// Must be overridden after database initialization in main.dart.
final localCategorySourceProvider = Provider<LocalCategorySource>((ref) {
  throw UnimplementedError('Must be overridden after DB init');
});

/// Provides the [SyncService] for background sync.
final syncServiceProvider = Provider<SyncService>((ref) {
  return SyncService(
    ref.read(localExpenseSourceProvider),
    ref.read(expenseRepositoryProvider),
  );
});
