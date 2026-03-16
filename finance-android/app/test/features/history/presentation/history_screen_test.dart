import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:finance_tracker/features/history/domain/filter_state.dart';
import 'package:finance_tracker/features/history/presentation/history_screen.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:finance_tracker/providers/expense_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

/// A fake [CategoryNotifier] for testing.
class FakeCategoryNotifier extends StateNotifier<CategoryState>
    implements CategoryNotifier {
  FakeCategoryNotifier(super.initial);

  @override
  Future<void> loadCategories() async {}

  @override
  Future<void> createCategory({
    required String name,
    required String icon,
    required String color,
  }) async {}

  @override
  Future<void> updateCategory({
    required String id,
    required String name,
    required String icon,
    required String color,
  }) async {}

  @override
  Future<void> deleteCategory(String id) async {}

  @override
  void reorderCategories(int oldIndex, int newIndex) {}

  @override
  Future<void> bulkCreateStarters() async {}
}

/// A fake [ExpenseNotifier] for testing.
class FakeExpenseNotifier extends StateNotifier<ExpenseState>
    implements ExpenseNotifier {
  FakeExpenseNotifier(super.initial);

  bool deleteExpenseCalled = false;
  String? lastDateFrom;
  String? lastDateTo;
  String? lastCategoryId;

  @override
  Future<bool> createExpense({
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async => true;

  @override
  Future<void> loadExpenses({
    int limit = 50,
    int offset = 0,
    String? dateFrom,
    String? dateTo,
    String? categoryId,
  }) async {
    lastDateFrom = dateFrom;
    lastDateTo = dateTo;
    lastCategoryId = categoryId;
  }

  @override
  Future<bool> updateExpense({
    required String id,
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async => true;

  @override
  Future<bool> deleteExpense(String id) async {
    deleteExpenseCalled = true;
    return true;
  }
}

const _testCategories = [
  Category(
    id: 'cat-1',
    name: 'Food',
    icon: 'restaurant',
    color: '#FF7043',
    sortOrder: 0,
  ),
  Category(
    id: 'cat-2',
    name: 'Transport',
    icon: 'directions_car',
    color: '#42A5F5',
    sortOrder: 1,
  ),
];

final _testExpenses = [
  Expense(
    id: 'exp-1',
    categoryId: 'cat-1',
    amountCents: 1250,
    note: 'Lunch',
    expenseDate: DateTime(2026, 3, 15),
  ),
  Expense(
    id: 'exp-2',
    categoryId: 'cat-1',
    amountCents: 500,
    note: '',
    expenseDate: DateTime(2026, 3, 14),
  ),
];

void main() {
  late FakeCategoryNotifier fakeCategoryNotifier;
  late FakeExpenseNotifier fakeExpenseNotifier;

  setUp(() {
    fakeCategoryNotifier = FakeCategoryNotifier(
      const CategoryLoaded(_testCategories),
    );
    fakeExpenseNotifier = FakeExpenseNotifier(
      ExpenseLoaded(_testExpenses),
    );
  });

  Widget buildSubject({FilterNotifier? filterNotifier}) {
    final router = GoRouter(
      initialLocation: '/history',
      routes: [
        GoRoute(
          path: '/history',
          builder: (_, __) => const Scaffold(
            body: HistoryScreen(),
          ),
        ),
        GoRoute(
          path: '/expenses/edit',
          builder: (_, state) => const Scaffold(
            body: Text('Edit Screen'),
          ),
        ),
      ],
    );
    return ProviderScope(
      overrides: [
        categoryStateProvider
            .overrideWith((_) => fakeCategoryNotifier),
        expenseStateProvider
            .overrideWith((_) => fakeExpenseNotifier),
        if (filterNotifier != null)
          filterStateProvider
              .overrideWith((_) => filterNotifier),
      ],
      child: MaterialApp.router(routerConfig: router),
    );
  }

  group('HistoryScreen', () {
    testWidgets('shows expense amounts', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(find.text(r'$12.50'), findsOneWidget);
      expect(find.text(r'$5.00'), findsOneWidget);
    });

    testWidgets('shows expense notes', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(find.text('Lunch'), findsOneWidget);
    });

    testWidgets('shows category chip for expenses', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // The restaurant icon should be rendered for category chips.
      expect(find.byIcon(Icons.restaurant), findsWidgets);
    });

    testWidgets('swipe to delete shows confirmation dialog',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Swipe the first expense row left.
      await tester.drag(
        find.text(r'$12.50'),
        const Offset(-500, 0),
      );
      await tester.pumpAndSettle();

      expect(
        find.text(r'Delete this $12.50 expense?'),
        findsOneWidget,
      );
    });

    testWidgets('cancel delete dismisses dialog', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Swipe to trigger dialog.
      await tester.drag(
        find.text(r'$12.50'),
        const Offset(-500, 0),
      );
      await tester.pumpAndSettle();

      // Tap "Keep Expense" to cancel.
      await tester.tap(find.text('Keep Expense'));
      await tester.pumpAndSettle();

      // Dialog should be gone and expense still visible.
      expect(
        find.text(r'Delete this $12.50 expense?'),
        findsNothing,
      );
      expect(find.text(r'$12.50'), findsOneWidget);
    });

    testWidgets('shows empty state when no expenses', (tester) async {
      fakeExpenseNotifier = FakeExpenseNotifier(
        const ExpenseLoaded([]),
      );

      final router = GoRouter(
        initialLocation: '/history',
        routes: [
          GoRoute(
            path: '/history',
            builder: (_, __) => const Scaffold(
              body: HistoryScreen(),
            ),
          ),
        ],
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            categoryStateProvider
                .overrideWith((_) => fakeCategoryNotifier),
            expenseStateProvider
                .overrideWith((_) => fakeExpenseNotifier),
          ],
          child: MaterialApp.router(routerConfig: router),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('No expenses yet'), findsOneWidget);
    });
  });

  group('Filter bar', () {
    testWidgets('filter bar shows All Dates and All Categories chips',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(find.text('All Dates'), findsOneWidget);
      expect(find.text('All Categories'), findsOneWidget);
    });

    testWidgets('tapping date chip opens date preset bottom sheet',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.tap(find.text('All Dates'));
      await tester.pumpAndSettle();

      expect(find.text('Filter by Date'), findsOneWidget);
      expect(find.text('Today'), findsOneWidget);
      expect(find.text('This Week'), findsOneWidget);
      expect(find.text('This Month'), findsOneWidget);
      expect(find.text('Last Month'), findsOneWidget);
      expect(find.text('Custom Range'), findsOneWidget);
    });

    testWidgets('selecting This Month preset updates chip label',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.tap(find.text('All Dates'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('This Month'));
      await tester.pumpAndSettle();

      expect(find.text('This Month'), findsOneWidget);
      expect(find.text('All Dates'), findsNothing);
    });

    testWidgets('tapping category chip opens category picker bottom sheet',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.tap(find.text('All Categories'));
      await tester.pumpAndSettle();

      expect(find.text('Filter by Category'), findsOneWidget);
      expect(find.text('Food'), findsWidgets);
      expect(find.text('Transport'), findsOneWidget);
    });

    testWidgets('selecting a category updates chip label',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.tap(find.text('All Categories'));
      await tester.pumpAndSettle();

      // Tap the Transport ListTile in the bottom sheet.
      await tester.tap(find.text('Transport'));
      await tester.pumpAndSettle();

      // Chip should show Transport now.
      expect(find.text('Transport'), findsOneWidget);
      expect(find.text('All Categories'), findsNothing);
    });

    testWidgets(
        'empty filter results show No expenses match these filters message',
        (tester) async {
      fakeExpenseNotifier = FakeExpenseNotifier(
        const ExpenseLoaded([]),
      );

      // Pre-set a filter so the empty state shows the filter message.
      final filterNotifier = FilterNotifier()
        ..setDatePreset(
          'Today',
          DateTime(2026, 3, 16),
          DateTime(2026, 3, 16),
        );

      final router = GoRouter(
        initialLocation: '/history',
        routes: [
          GoRoute(
            path: '/history',
            builder: (_, __) => const Scaffold(
              body: HistoryScreen(),
            ),
          ),
        ],
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            categoryStateProvider
                .overrideWith((_) => fakeCategoryNotifier),
            expenseStateProvider
                .overrideWith((_) => fakeExpenseNotifier),
            filterStateProvider
                .overrideWith((_) => filterNotifier),
          ],
          child: MaterialApp.router(routerConfig: router),
        ),
      );
      await tester.pumpAndSettle();

      expect(
        find.text('No expenses match these filters'),
        findsOneWidget,
      );
      expect(
        find.text('Try adjusting your date range or category.'),
        findsOneWidget,
      );
    });
  });
}
