import 'package:finance_tracker/core/database/local_expense_source.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

void main() {
  late Database db;
  late LocalExpenseSource source;

  setUpAll(() {
    sqfliteFfiInit();
    databaseFactory = databaseFactoryFfi;
  });

  setUp(() async {
    db = await openDatabase(
      inMemoryDatabasePath,
      version: 1,
      onCreate: (db, version) async {
        await db.execute('''
          CREATE TABLE local_expenses (
            id TEXT PRIMARY KEY,
            server_id TEXT,
            category_id TEXT NOT NULL,
            amount_cents INTEGER NOT NULL,
            note TEXT NOT NULL DEFAULT '',
            expense_date TEXT NOT NULL,
            synced INTEGER NOT NULL DEFAULT 0,
            created_at TEXT NOT NULL DEFAULT (datetime('now'))
          )
        ''');
      },
    );
    source = LocalExpenseSource(db);
  });

  tearDown(() async {
    await db.close();
  });

  Expense _makeExpense({
    String id = 'exp-1',
    String categoryId = 'cat-1',
    int amountCents = 1500,
    String note = 'lunch',
    DateTime? expenseDate,
  }) {
    return Expense(
      id: id,
      categoryId: categoryId,
      amountCents: amountCents,
      note: note,
      expenseDate: expenseDate ?? DateTime(2026, 3, 15),
    );
  }

  group('LocalExpenseSource', () {
    test('insertExpense stores a row in local_expenses', () async {
      await source.insertExpense(_makeExpense());

      final rows = await db.query('local_expenses');
      expect(rows, hasLength(1));
      expect(rows.first['id'], 'exp-1');
      expect(rows.first['category_id'], 'cat-1');
      expect(rows.first['amount_cents'], 1500);
      expect(rows.first['note'], 'lunch');
      expect(rows.first['synced'], 0);
    });

    test('insertExpense with synced=true sets synced to 1', () async {
      await source.insertExpense(_makeExpense(), synced: true);

      final rows = await db.query('local_expenses');
      expect(rows.first['synced'], 1);
    });

    test('getUnsyncedExpenses returns only unsynced in FIFO order', () async {
      // Insert older expense first.
      await source.insertExpense(_makeExpense(id: 'exp-old'));
      // Small delay so created_at differs.
      await Future<void>.delayed(const Duration(milliseconds: 10));
      await source.insertExpense(_makeExpense(id: 'exp-new'));
      // Insert a synced one that should not appear.
      await source.insertExpense(
        _makeExpense(id: 'exp-synced'),
        synced: true,
      );

      final unsynced = await source.getUnsyncedExpenses();
      expect(unsynced, hasLength(2));
      expect(unsynced[0]['id'], 'exp-old');
      expect(unsynced[1]['id'], 'exp-new');
    });

    test('markSynced updates synced flag and server_id', () async {
      await source.insertExpense(_makeExpense());

      await source.markSynced('exp-1', 'server-abc');

      final rows = await db.query('local_expenses');
      expect(rows.first['synced'], 1);
      expect(rows.first['server_id'], 'server-abc');
    });

    test('getAllExpenses returns all rows', () async {
      await source.insertExpense(_makeExpense(id: 'a'));
      await source.insertExpense(_makeExpense(id: 'b'), synced: true);

      final all = await source.getAllExpenses();
      expect(all, hasLength(2));
    });

    test('getUnsyncedCount returns correct count', () async {
      await source.insertExpense(_makeExpense(id: 'a'));
      await source.insertExpense(_makeExpense(id: 'b'));
      await source.insertExpense(_makeExpense(id: 'c'), synced: true);

      final count = await source.getUnsyncedCount();
      expect(count, 2);
    });
  });
}
