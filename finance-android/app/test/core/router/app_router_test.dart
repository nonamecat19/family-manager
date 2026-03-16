import 'package:finance_tracker/core/router/app_router.dart';
import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:finance_tracker/providers/auth_provider.dart';
import 'package:finance_tracker/providers/expense_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

/// A fake [AuthNotifier] for testing that starts with a given state.
class FakeAuthNotifier extends StateNotifier<AuthState>
    implements AuthNotifier {
  FakeAuthNotifier(super.initialState);

  @override
  Future<void> login(String email, String password) async {}

  @override
  Future<void> signup(String email, String password) async {}

  @override
  Future<void> logout() async {}

  @override
  Future<void> tryRestoreSession() async {}
}

/// A fake [ExpenseNotifier] for testing that stays in loaded state.
class FakeExpenseNotifier extends StateNotifier<ExpenseState>
    implements ExpenseNotifier {
  FakeExpenseNotifier() : super(const ExpenseLoaded([]));

  @override
  Future<bool> createExpense({
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async => true;

  @override
  Future<void> loadExpenses({int limit = 50, int offset = 0, String? dateFrom, String? dateTo, String? categoryId}) async {}

  @override
  Future<bool> updateExpense({
    required String id,
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async => true;

  @override
  Future<bool> deleteExpense(String id) async => true;
}

void main() {
  group('AppRouter', () {
    testWidgets(
      'redirects unauthenticated user to /welcome',
      (tester) async {
        final notifier = FakeAuthNotifier(const Unauthenticated());

        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              authStateProvider.overrideWith((_) => notifier),
            ],
            child: Consumer(
              builder: (context, ref, _) {
                final router = ref.watch(goRouterProvider);
                return MaterialApp.router(routerConfig: router);
              },
            ),
          ),
        );
        await tester.pumpAndSettle();

        // Should show WelcomeScreen because user is unauthenticated.
        expect(find.text('FinanceTracker'), findsOneWidget);
        expect(find.text('Log In'), findsOneWidget);
        expect(find.text('Sign Up'), findsOneWidget);
      },
    );

    testWidgets(
      'allows authenticated user to access /history',
      (tester) async {
        final notifier = FakeAuthNotifier(
          const Authenticated(userId: 'test-id', email: 'test@test.com'),
        );

        final fakeExpenseNotifier = FakeExpenseNotifier();

        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              authStateProvider.overrideWith((_) => notifier),
              expenseStateProvider
                  .overrideWith((_) => fakeExpenseNotifier),
            ],
            child: Consumer(
              builder: (context, ref, _) {
                final router = ref.watch(goRouterProvider);
                return MaterialApp.router(routerConfig: router);
              },
            ),
          ),
        );
        await tester.pumpAndSettle();

        // Should show History screen with empty expense state.
        expect(find.text('No expenses yet'), findsOneWidget);
      },
    );
  });
}
