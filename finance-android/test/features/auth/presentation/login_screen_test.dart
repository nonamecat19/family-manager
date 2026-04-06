import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:finance_tracker/features/auth/presentation/login_screen.dart';
import 'package:finance_tracker/providers/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

/// A fake [AuthNotifier] for testing that doesn't need real deps.
class FakeAuthNotifier extends StateNotifier<AuthState>
    implements AuthNotifier {
  FakeAuthNotifier() : super(const Unauthenticated());

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
  late FakeAuthNotifier fakeAuthNotifier;

  setUp(() {
    fakeAuthNotifier = FakeAuthNotifier();
  });

  Widget buildSubject() {
    final router = GoRouter(
      initialLocation: '/login',
      routes: [
        GoRoute(
          path: '/login',
          builder: (_, __) => const LoginScreen(),
        ),
        GoRoute(
          path: '/welcome',
          builder: (_, __) => const Scaffold(body: Text('Welcome')),
        ),
      ],
    );

    return ProviderScope(
      overrides: [
        authStateProvider
            .overrideWith((_) => fakeAuthNotifier),
      ],
      child: MaterialApp.router(routerConfig: router),
    );
  }

  group('LoginScreen', () {
    testWidgets('renders email and password fields', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(find.text('Email'), findsOneWidget);
      expect(find.text('Password'), findsOneWidget);
      expect(find.text('Log In'), findsWidgets); // Button + AppBar title
    });

    testWidgets('shows validation error for empty email', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Tap Log In button without entering anything.
      await tester.tap(find.widgetWithText(FilledButton, 'Log In'));
      await tester.pumpAndSettle();

      expect(find.text('Email is required'), findsOneWidget);
    });

    testWidgets('shows validation error for invalid email', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.enterText(
        find.widgetWithText(TextFormField, 'Email'),
        'notanemail',
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Log In'));
      await tester.pumpAndSettle();

      expect(find.text('Enter a valid email address'), findsOneWidget);
    });

    testWidgets('shows validation error for short password', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.enterText(
        find.widgetWithText(TextFormField, 'Email'),
        'test@example.com',
      );
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Password'),
        'short',
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Log In'));
      await tester.pumpAndSettle();

      expect(
        find.text('Password must be at least 8 characters'),
        findsOneWidget,
      );
    });
  });
}
