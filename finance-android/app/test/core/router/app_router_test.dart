import 'package:finance_tracker/core/router/app_router.dart';
import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:finance_tracker/providers/auth_provider.dart';
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

        // Should show History screen (the initial authenticated location).
        expect(find.text('History'), findsWidgets);
      },
    );
  });
}
