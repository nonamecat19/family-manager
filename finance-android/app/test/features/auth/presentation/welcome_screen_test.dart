import 'package:finance_tracker/features/auth/presentation/welcome_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  group('WelcomeScreen', () {
    testWidgets('renders Log In and Sign Up buttons', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: WelcomeScreen(),
        ),
      );

      expect(find.text('Log In'), findsOneWidget);
      expect(find.text('Sign Up'), findsOneWidget);
      expect(find.text('FinanceTracker'), findsOneWidget);
    });

    testWidgets('tapping Log In navigates to /login', (tester) async {
      String? navigatedTo;

      final router = GoRouter(
        initialLocation: '/welcome',
        routes: [
          GoRoute(
            path: '/welcome',
            builder: (_, __) => const WelcomeScreen(),
          ),
          GoRoute(
            path: '/login',
            builder: (_, __) {
              navigatedTo = '/login';
              return const Scaffold(body: Text('Login'));
            },
          ),
          GoRoute(
            path: '/signup',
            builder: (_, __) => const Scaffold(body: Text('Signup')),
          ),
        ],
      );

      await tester.pumpWidget(
        MaterialApp.router(routerConfig: router),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Log In'));
      await tester.pumpAndSettle();

      expect(navigatedTo, equals('/login'));
    });

    testWidgets('tapping Sign Up navigates to /signup', (tester) async {
      String? navigatedTo;

      final router = GoRouter(
        initialLocation: '/welcome',
        routes: [
          GoRoute(
            path: '/welcome',
            builder: (_, __) => const WelcomeScreen(),
          ),
          GoRoute(
            path: '/login',
            builder: (_, __) => const Scaffold(body: Text('Login')),
          ),
          GoRoute(
            path: '/signup',
            builder: (_, __) {
              navigatedTo = '/signup';
              return const Scaffold(body: Text('Signup'));
            },
          ),
        ],
      );

      await tester.pumpWidget(
        MaterialApp.router(routerConfig: router),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Sign Up'));
      await tester.pumpAndSettle();

      expect(navigatedTo, equals('/signup'));
    });
  });
}
