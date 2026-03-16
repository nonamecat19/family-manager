import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:sqflite/sqflite.dart';

/// Local CRUD operations for expenses stored in sqflite.
///
/// Manages the sync queue: new expenses are inserted as unsynced,
/// then marked synced after successful server POST.
class LocalExpenseSource {
  /// Creates a [LocalExpenseSource] with the given [Database].
  const LocalExpenseSource(this._db);

  final Database _db;

  /// Inserts an expense into the local database.
  ///
  /// [synced] defaults to false (pending sync).
  Future<void> insertExpense(Expense expense, {bool synced = false}) async {
    await _db.insert(
      'local_expenses',
      {
        'id': expense.id,
        'category_id': expense.categoryId,
        'amount_cents': expense.amountCents,
        'note': expense.note,
        'expense_date':
            '${expense.expenseDate.year.toString().padLeft(4, '0')}-'
            '${expense.expenseDate.month.toString().padLeft(2, '0')}-'
            '${expense.expenseDate.day.toString().padLeft(2, '0')}',
        'synced': synced ? 1 : 0,
        'created_at': DateTime.now().toIso8601String(),
      },
      conflictAlgorithm: ConflictAlgorithm.replace,
    );
  }

  /// Returns all unsynced expenses ordered by created_at ASC (FIFO).
  Future<List<Map<String, dynamic>>> getUnsyncedExpenses() async {
    return _db.query(
      'local_expenses',
      where: 'synced = ?',
      whereArgs: [0],
      orderBy: 'created_at ASC',
    );
  }

  /// Marks an expense as synced with the server-assigned ID.
  Future<void> markSynced(String localId, String serverId) async {
    await _db.update(
      'local_expenses',
      {'synced': 1, 'server_id': serverId},
      where: 'id = ?',
      whereArgs: [localId],
    );
  }

  /// Returns all local expenses ordered by date descending.
  Future<List<Map<String, dynamic>>> getAllExpenses() async {
    return _db.query(
      'local_expenses',
      orderBy: 'expense_date DESC, created_at DESC',
    );
  }

  /// Returns the count of unsynced expenses.
  Future<int> getUnsyncedCount() async {
    final result = await _db.rawQuery(
      'SELECT COUNT(*) as count FROM local_expenses WHERE synced = 0',
    );
    return Sqflite.firstIntValue(result) ?? 0;
  }
}
