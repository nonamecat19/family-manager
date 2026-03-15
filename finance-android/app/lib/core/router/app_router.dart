import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:finance_tracker/features/auth/presentation/login_screen.dart';
import 'package:finance_tracker/features/auth/presentation/signup_screen.dart';
import 'package:finance_tracker/features/auth/presentation/welcome_screen.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart'
    as models;
import 'package:finance_tracker/features/categories/presentation/categories_screen.dart';
import 'package:finance_tracker/features/categories/presentation/category_form_screen.dart';
import 'package:finance_tracker/features/charts/presentation/charts_screen.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/expenses/presentation/expense_form_screen.dart';
import 'package:finance_tracker/features/history/presentation/history_screen.dart';
import 'package:finance_tracker/features/settings/presentation/settings_screen.dart';
import 'package:finance_tracker/providers/auth_provider.dart';
import 'package:finance_tracker/shared/widgets/app_scaffold.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// Auth routes that don't require authentication.
const _authRoutes = {'/welcome', '/login', '/signup'};

/// Provides the app's [GoRouter] with auth-aware redirect logic.
///
/// Watches [authStateProvider] and rebuilds the router when auth
/// state changes, redirecting unauthenticated users to the welcome
/// screen and authenticated users away from auth routes.
final goRouterProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authStateProvider);

  return GoRouter(
    initialLocation: '/history',
    redirect: (context, state) {
      // While restoring session, don't redirect.
      if (authState is AuthInitial || authState is AuthLoading) {
        return null;
      }

      final isAuthenticated = authState is Authenticated;
      final isAuthRoute = _authRoutes.contains(state.matchedLocation);

      if (!isAuthenticated && !isAuthRoute) return '/welcome';
      if (isAuthenticated && isAuthRoute) return '/history';
      return null;
    },
    routes: [
      GoRoute(
        path: '/welcome',
        builder: (context, state) => const WelcomeScreen(),
      ),
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/signup',
        builder: (context, state) => const SignupScreen(),
      ),
      ShellRoute(
        builder: (context, state, child) {
          final index = switch (state.uri.path) {
            '/history' => 0,
            '/charts' => 1,
            '/settings' => 2,
            _ => 0,
          };
          return AppScaffold(
            selectedIndex: index,
            child: child,
          );
        },
        routes: [
          GoRoute(
            path: '/history',
            builder: (context, state) => const HistoryScreen(),
          ),
          GoRoute(
            path: '/charts',
            builder: (context, state) => const ChartsScreen(),
          ),
          GoRoute(
            path: '/settings',
            builder: (context, state) => const SettingsScreen(),
          ),
        ],
      ),
      // Expense entry route (outside ShellRoute -- own AppBar, no bottom nav).
      GoRoute(
        path: '/expenses/new',
        builder: (context, state) => const ExpenseFormScreen(),
      ),
      GoRoute(
        path: '/expenses/edit',
        builder: (context, state) => ExpenseFormScreen(
          expense: state.extra as Expense?,
        ),
      ),
      // Category management routes (outside ShellRoute -- own AppBar,
      // no bottom nav).
      GoRoute(
        path: '/settings/categories',
        builder: (context, state) => const CategoriesScreen(),
      ),
      GoRoute(
        path: '/settings/categories/new',
        builder: (context, state) => const CategoryFormScreen(),
      ),
      GoRoute(
        path: '/settings/categories/edit',
        builder: (context, state) => CategoryFormScreen(
          category: state.extra as models.Category?,
        ),
      ),
    ],
  );
});
