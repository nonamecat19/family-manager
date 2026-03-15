import 'package:finance_tracker/features/categories/data/category_repository.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:finance_tracker/features/expenses/presentation/expense_form_screen.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:finance_tracker/providers/expense_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:mocktail/mocktail.dart';

class MockCategoryRepository extends Mock implements CategoryRepository {}

/// A fake [CategoryNotifier] for testing.
class FakeCategoryNotifier extends StateNotifier<CategoryState>
    implements CategoryNotifier {
  FakeCategoryNotifier([CategoryState initial = const CategoryInitial()])
      : super(initial);

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
  FakeExpenseNotifier() : super(const ExpenseInitial());

  bool createExpenseCalled = false;

  @override
  Future<bool> createExpense({
    required String categoryId,
    required int amountCents,
    required String note,
    required String expenseDate,
  }) async {
    createExpenseCalled = true;
    return true;
  }

  @override
  Future<void> loadExpenses({int limit = 50, int offset = 0}) async {}
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

void main() {
  late FakeCategoryNotifier fakeCategoryNotifier;
  late FakeExpenseNotifier fakeExpenseNotifier;

  setUp(() {
    fakeCategoryNotifier = FakeCategoryNotifier(
      const CategoryLoaded(_testCategories),
    );
    fakeExpenseNotifier = FakeExpenseNotifier();
  });

  Widget buildSubject() {
    final router = GoRouter(
      initialLocation: '/expenses/new',
      routes: [
        GoRoute(
          path: '/expenses/new',
          builder: (_, __) => const ExpenseFormScreen(),
        ),
      ],
    );

    return ProviderScope(
      overrides: [
        categoryStateProvider.overrideWith((_) => fakeCategoryNotifier),
        expenseStateProvider.overrideWith((_) => fakeExpenseNotifier),
      ],
      child: MaterialApp.router(routerConfig: router),
    );
  }

  group('ExpenseFormScreen', () {
    testWidgets('renders all four fields: amount, category, note, date',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Amount field
      expect(find.text('Amount'), findsOneWidget);
      // Category section
      expect(find.text('Category'), findsOneWidget);
      // Note field
      expect(find.text('Note (optional)'), findsOneWidget);
      // Date field shows "Today"
      expect(find.text('Today'), findsOneWidget);
    });

    testWidgets('amount field has autofocus', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      final textField = tester.widget<TextField>(
        find.byType(TextField).first,
      );
      expect(textField.autofocus, isTrue);
    });

    testWidgets('amount field has numpad keyboard type', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      final textField = tester.widget<TextField>(
        find.byType(TextField).first,
      );
      expect(
        textField.keyboardType,
        const TextInputType.numberWithOptions(decimal: true),
      );
    });

    testWidgets('date shows today by default', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(find.text('Today'), findsOneWidget);
    });

    testWidgets('shows category chips from loaded categories',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(find.text('Food'), findsOneWidget);
      expect(find.text('Transport'), findsOneWidget);
    });

    testWidgets('save with empty amount shows error', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(FilledButton, 'Save'));
      await tester.pumpAndSettle();

      expect(find.text('Amount is required'), findsOneWidget);
    });

    testWidgets('save without category selected shows error',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Enter a valid amount but don't select category.
      await tester.enterText(find.byType(TextField).first, '12.34');
      await tester.tap(find.widgetWithText(FilledButton, 'Save'));
      await tester.pumpAndSettle();

      expect(find.text('Select a category'), findsOneWidget);
    });

    testWidgets('selecting a category chip highlights it', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Tap the "Food" chip.
      await tester.tap(find.text('Food'));
      await tester.pumpAndSettle();

      // The chip should now have a visible border (selected state).
      // We verify by finding the GestureDetector wrapping Food and checking
      // that the Container has a non-transparent border.
      final containers = tester.widgetList<Container>(
        find.descendant(
          of: find.byType(GestureDetector),
          matching: find.byType(Container),
        ),
      );

      // At least one container should have a border with non-transparent color.
      final hasSelectedBorder = containers.any((container) {
        final decoration = container.decoration;
        if (decoration is BoxDecoration && decoration.border != null) {
          final border = decoration.border;
          if (border is Border) {
            return border.top.color != Colors.transparent &&
                border.top.width == 2;
          }
        }
        return false;
      });
      expect(hasSelectedBorder, isTrue);
    });
  });
}
