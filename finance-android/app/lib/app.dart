import 'package:finance_tracker/core/router/app_router.dart';
import 'package:finance_tracker/core/theme/app_theme.dart';
import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:finance_tracker/providers/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Root widget of the FinanceTracker application.
///
/// Sets up [MaterialApp.router] with light and dark themes,
/// system-driven theme mode, and auth-aware routing.
class FinanceApp extends ConsumerWidget {
  /// Creates a [FinanceApp].
  const FinanceApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authStateProvider);
    final router = ref.watch(goRouterProvider);

    // Show loading screen while restoring session.
    if (authState is AuthInitial || authState is AuthLoading) {
      return MaterialApp(
        title: 'FinanceTracker',
        theme: buildAppTheme(Brightness.light),
        darkTheme: buildAppTheme(Brightness.dark),
        home: const Scaffold(
          body: Center(
            child: CircularProgressIndicator(),
          ),
        ),
      );
    }

    return MaterialApp.router(
      title: 'FinanceTracker',
      theme: buildAppTheme(Brightness.light),
      darkTheme: buildAppTheme(Brightness.dark),
      routerConfig: router,
    );
  }
}
