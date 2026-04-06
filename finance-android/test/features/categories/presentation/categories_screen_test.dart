import 'package:finance_tracker/features/categories/data/category_repository.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:finance_tracker/features/categories/presentation/categories_screen.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:mocktail/mocktail.dart';
import 'package:shared_preferences/shared_preferences.dart';

class MockCategoryRepository extends Mock implements CategoryRepository {}

/// A fake [CategoryNotifier] for testing that doesn't need real deps.
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

void main() {
  late FakeCategoryNotifier fakeCategoryNotifier;

  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  Widget buildSubject(FakeCategoryNotifier notifier) {
    final router = GoRouter(
      initialLocation: '/settings/categories',
      routes: [
        GoRoute(
          path: '/settings/categories',
          builder: (_, __) => const CategoriesScreen(),
        ),
        GoRoute(
          path: '/settings/categories/new',
          builder: (_, __) => const Scaffold(body: Text('New Form')),
        ),
        GoRoute(
          path: '/settings/categories/edit',
          builder: (_, state) => Scaffold(
            body: Text('Edit ${(state.extra as Category?)?.name}'),
          ),
        ),
      ],
    );

    return ProviderScope(
      overrides: [
        categoryStateProvider.overrideWith((_) => notifier),
      ],
      child: MaterialApp.router(routerConfig: router),
    );
  }

  group('CategoriesScreen', () {
    testWidgets('shows loading indicator during CategoryLoading',
        (tester) async {
      fakeCategoryNotifier =
          FakeCategoryNotifier(const CategoryLoading());
      await tester.pumpWidget(buildSubject(fakeCategoryNotifier));
      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('renders category list when loaded state has categories',
        (tester) async {
      fakeCategoryNotifier = FakeCategoryNotifier(
        const CategoryLoaded([
          Category(
            id: '1',
            name: 'Food',
            icon: 'restaurant',
            color: '#FF7043',
            sortOrder: 0,
          ),
          Category(
            id: '2',
            name: 'Transport',
            icon: 'directions_car',
            color: '#42A5F5',
            sortOrder: 1,
          ),
        ]),
      );
      await tester.pumpWidget(buildSubject(fakeCategoryNotifier));
      await tester.pumpAndSettle();

      expect(find.text('Food'), findsOneWidget);
      expect(find.text('Transport'), findsOneWidget);
    });

    testWidgets('shows starter prompt when no categories and not dismissed',
        (tester) async {
      SharedPreferences.setMockInitialValues({});
      fakeCategoryNotifier =
          FakeCategoryNotifier(const CategoryLoaded([]));
      await tester.pumpWidget(buildSubject(fakeCategoryNotifier));
      await tester.pumpAndSettle();

      expect(find.text('Get started quickly?'), findsOneWidget);
      expect(find.text('Add these'), findsOneWidget);
      expect(find.text('Skip'), findsOneWidget);
    });

    testWidgets(
        'shows empty message when no categories and starter dismissed',
        (tester) async {
      SharedPreferences.setMockInitialValues(
          {'categories_starter_dismissed': true});
      fakeCategoryNotifier =
          FakeCategoryNotifier(const CategoryLoaded([]));
      await tester.pumpWidget(buildSubject(fakeCategoryNotifier));
      await tester.pumpAndSettle();

      expect(
        find.text('No categories yet. Tap + to create one.'),
        findsOneWidget,
      );
    });

    testWidgets('shows error message on CategoryError', (tester) async {
      fakeCategoryNotifier =
          FakeCategoryNotifier(const CategoryError('Network error'));
      await tester.pumpWidget(buildSubject(fakeCategoryNotifier));
      await tester.pump();

      expect(find.text('Network error'), findsOneWidget);
    });
  });
}
