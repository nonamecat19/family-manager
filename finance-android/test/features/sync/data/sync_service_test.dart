import 'package:dio/dio.dart';
import 'package:finance_tracker/core/database/local_expense_source.dart';
import 'package:finance_tracker/features/expenses/data/expense_repository.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/sync/data/sync_service.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

class MockLocalExpenseSource extends Mock implements LocalExpenseSource {}

class MockExpenseRepository extends Mock implements ExpenseRepository {}

void main() {
  late MockLocalExpenseSource mockLocalSource;
  late MockExpenseRepository mockRepository;
  late SyncService syncService;

  setUp(() {
    mockLocalSource = MockLocalExpenseSource();
    mockRepository = MockExpenseRepository();
    syncService = SyncService(mockLocalSource, mockRepository);
  });

  Map<String, dynamic> _makeRow({
    required String id,
    String categoryId = 'cat-1',
    int amountCents = 1500,
    String note = 'lunch',
    String expenseDate = '2026-03-15',
  }) {
    return {
      'id': id,
      'category_id': categoryId,
      'amount_cents': amountCents,
      'note': note,
      'expense_date': expenseDate,
      'synced': 0,
      'created_at': '2026-03-15T12:00:00.000',
    };
  }

  Expense _makeServerExpense(String id) {
    return Expense(
      id: id,
      categoryId: 'cat-1',
      amountCents: 1500,
      note: 'lunch',
      expenseDate: DateTime(2026, 3, 15),
    );
  }

  group('SyncService', () {
    test('syncs pending expenses to server', () async {
      when(() => mockLocalSource.getUnsyncedExpenses()).thenAnswer(
        (_) async => [_makeRow(id: 'local-1'), _makeRow(id: 'local-2')],
      );
      when(
        () => mockRepository.createExpense(
          categoryId: any(named: 'categoryId'),
          amountCents: any(named: 'amountCents'),
          note: any(named: 'note'),
          expenseDate: any(named: 'expenseDate'),
        ),
      ).thenAnswer((_) async => _makeServerExpense('server-1'));
      when(
        () => mockLocalSource.markSynced(any(), any()),
      ).thenAnswer((_) async {});

      final result = await syncService.syncPendingExpenses();

      expect(result.syncedCount, 2);
      expect(result.failedCount, 0);
      expect(result.hasFailures, isFalse);
      verify(() => mockLocalSource.markSynced('local-1', 'server-1'))
          .called(1);
      verify(() => mockLocalSource.markSynced('local-2', 'server-1'))
          .called(1);
    });

    test('stops on DioException and reports partial result', () async {
      when(() => mockLocalSource.getUnsyncedExpenses()).thenAnswer(
        (_) async => [_makeRow(id: 'local-1'), _makeRow(id: 'local-2')],
      );

      var callCount = 0;
      when(
        () => mockRepository.createExpense(
          categoryId: any(named: 'categoryId'),
          amountCents: any(named: 'amountCents'),
          note: any(named: 'note'),
          expenseDate: any(named: 'expenseDate'),
        ),
      ).thenAnswer((_) async {
        callCount++;
        if (callCount == 1) return _makeServerExpense('server-1');
        throw DioException(requestOptions: RequestOptions());
      });
      when(
        () => mockLocalSource.markSynced(any(), any()),
      ).thenAnswer((_) async {});

      final result = await syncService.syncPendingExpenses();

      expect(result.syncedCount, 1);
      expect(result.failedCount, 1);
      expect(result.hasFailures, isTrue);
      verify(() => mockLocalSource.markSynced('local-1', 'server-1'))
          .called(1);
      verifyNever(() => mockLocalSource.markSynced('local-2', any()));
    });

    test('returns empty result when no pending expenses', () async {
      when(() => mockLocalSource.getUnsyncedExpenses())
          .thenAnswer((_) async => []);

      final result = await syncService.syncPendingExpenses();

      expect(result.syncedCount, 0);
      expect(result.failedCount, 0);
    });

    test('prevents concurrent syncs via mutex', () async {
      // Make getUnsyncedExpenses slow so first call is still running.
      when(() => mockLocalSource.getUnsyncedExpenses()).thenAnswer(
        (_) async {
          await Future<void>.delayed(const Duration(milliseconds: 100));
          return [_makeRow(id: 'local-1')];
        },
      );
      when(
        () => mockRepository.createExpense(
          categoryId: any(named: 'categoryId'),
          amountCents: any(named: 'amountCents'),
          note: any(named: 'note'),
          expenseDate: any(named: 'expenseDate'),
        ),
      ).thenAnswer((_) async => _makeServerExpense('server-1'));
      when(
        () => mockLocalSource.markSynced(any(), any()),
      ).thenAnswer((_) async {});

      // Fire both calls without awaiting.
      final results = await Future.wait([
        syncService.syncPendingExpenses(),
        syncService.syncPendingExpenses(),
      ]);

      // Second call should return empty result due to mutex.
      expect(results[1].syncedCount, 0);
      // getUnsyncedExpenses should only be called once (mutex blocked second).
      verify(() => mockLocalSource.getUnsyncedExpenses()).called(1);
    });
  });
}
