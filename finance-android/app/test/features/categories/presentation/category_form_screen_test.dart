import 'package:finance_tracker/features/categories/data/category_repository.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:finance_tracker/features/categories/presentation/category_form_screen.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:mocktail/mocktail.dart';

class MockCategoryRepository extends Mock implements CategoryRepository {}

/// A fake [CategoryNotifier] for testing.
class FakeCategoryNotifier extends StateNotifier<CategoryState>
    implements CategoryNotifier {
  FakeCategoryNotifier() : super(const CategoryLoaded([]));

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

void main() {
  late FakeCategoryNotifier fakeCategoryNotifier;

  setUp(() {
    fakeCategoryNotifier = FakeCategoryNotifier();
  });

  Widget buildSubject({Category? category}) {
    final router = GoRouter(
      initialLocation: '/form',
      routes: [
        GoRoute(
          path: '/form',
          builder: (_, __) => CategoryFormScreen(category: category),
        ),
      ],
    );

    return ProviderScope(
      overrides: [
        categoryStateProvider.overrideWith((_) => fakeCategoryNotifier),
      ],
      child: MaterialApp.router(routerConfig: router),
    );
  }

  group('CategoryFormScreen', () {
    testWidgets('shows "New Category" title in create mode',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(find.text('New Category'), findsOneWidget);
    });

    testWidgets('shows "Edit Category" title and pre-filled name in edit mode',
        (tester) async {
      const category = Category(
        id: '1',
        name: 'Food',
        icon: 'restaurant',
        color: '#FF7043',
        sortOrder: 0,
      );
      await tester.pumpWidget(buildSubject(category: category));
      await tester.pumpAndSettle();

      expect(find.text('Edit Category'), findsOneWidget);
      // The name field should contain the pre-filled text.
      final nameField = tester.widget<TextFormField>(
        find.byType(TextFormField),
      );
      expect(nameField.controller?.text, 'Food');
    });

    testWidgets('validates required name field on submit', (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Tap Create Category without filling anything.
      await tester.tap(
        find.widgetWithText(FilledButton, 'Create Category'),
      );
      await tester.pumpAndSettle();

      expect(find.text('Name is required'), findsOneWidget);
    });

    testWidgets('validates icon selection required on submit',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Fill name but no icon or color.
      await tester.enterText(find.byType(TextFormField), 'Test');
      await tester.tap(
        find.widgetWithText(FilledButton, 'Create Category'),
      );
      await tester.pumpAndSettle();

      expect(find.text('Icon is required'), findsOneWidget);
    });

    testWidgets('validates color selection required on submit',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      // Fill name but no icon or color -- both errors show.
      await tester.enterText(find.byType(TextFormField), 'Test');
      await tester.tap(
        find.widgetWithText(FilledButton, 'Create Category'),
      );
      await tester.pumpAndSettle();

      expect(find.text('Color is required'), findsOneWidget);
    });

    testWidgets('shows "Save Changes" button in edit mode', (tester) async {
      const category = Category(
        id: '1',
        name: 'Food',
        icon: 'restaurant',
        color: '#FF7043',
        sortOrder: 0,
      );
      await tester.pumpWidget(buildSubject(category: category));
      await tester.pumpAndSettle();

      expect(
        find.widgetWithText(FilledButton, 'Save Changes'),
        findsOneWidget,
      );
    });

    testWidgets('shows "Create Category" button in create mode',
        (tester) async {
      await tester.pumpWidget(buildSubject());
      await tester.pumpAndSettle();

      expect(
        find.widgetWithText(FilledButton, 'Create Category'),
        findsOneWidget,
      );
    });
  });
}
