import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:finance_tracker/features/auth/presentation/signup_screen.dart';
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
      initialLocation: '/signup',
      routes: [
        GoRoute(
          path: '/signup',
          builder: (_, __) => const SignupScreen(),
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

  group('SignupScreen', () {
    testWidgets('renders email and password fields', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(find.text('Email'), findsOneWidget);
      expect(find.text('Password'), findsOneWidget);
      expect(find.text('Sign Up'), findsWidgets); // Button + AppBar title
    });

    testWidgets('shows validation error for empty email', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(FilledButton, 'Sign Up'));
      await tester.pumpAndSettle();

      expect(find.text('Email is required'), findsOneWidget);
    });

    testWidgets('shows validation error for invalid email', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.enterText(
        find.widgetWithText(TextFormField, 'Email'),
        'bademail',
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Sign Up'));
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
        'abc',
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Sign Up'));
      await tester.pumpAndSettle();

      expect(
        find.text('Password must be at least 8 characters'),
        findsOneWidget,
      );
    });
  });
}
