import 'package:finance_tracker/features/family/data/models/family_expense.dart';
import 'package:finance_tracker/features/family/domain/family_feed_notifier.dart';
import 'package:finance_tracker/features/family/presentation/family_feed_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Widget buildSubject(FakeFamilyFeedNotifier notifier) {
    return ProviderScope(
      overrides: [
        familyFeedStateProvider.overrideWith((_) => notifier),
      ],
      child: const MaterialApp(home: FamilyFeedScreen()),
    );
  }

  group('FamilyFeedScreen', () {
    testWidgets(
      'shows loading indicator for FamilyFeedLoading state',
      (tester) async {
        final notifier =
            FakeFamilyFeedNotifier(const FamilyFeedLoading());
        await tester.pumpWidget(buildSubject(notifier));
        await tester.pump();

        expect(
          find.byType(CircularProgressIndicator),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'shows expense with user email for FamilyFeedLoaded state',
      (tester) async {
        final testExpense = FamilyExpense(
          id: 'e1',
          userId: 'u1',
          userEmail: 'alice@test.com',
          categoryId: 'c1',
          categoryName: 'Food',
          categoryColor: '#FF7043',
          categoryIcon: 'restaurant',
          amountCents: 1500,
          note: 'Lunch',
          expenseDate: DateTime(2026, 3, 10),
        );
        final notifier = FakeFamilyFeedNotifier(
          FamilyFeedLoaded([testExpense]),
        );
        await tester.pumpWidget(buildSubject(notifier));
        await tester.pump();

        expect(find.text('alice@test.com'), findsOneWidget);
        expect(find.text(r'$15.00'), findsOneWidget);
        expect(find.text('Lunch'), findsOneWidget);
      },
    );

    testWidgets(
      'shows empty state when no expenses',
      (tester) async {
        final notifier = FakeFamilyFeedNotifier(
          const FamilyFeedLoaded([]),
        );
        await tester.pumpWidget(buildSubject(notifier));
        await tester.pump();

        expect(find.text('No family expenses'), findsOneWidget);
      },
    );

    testWidgets(
      'shows error message for FamilyFeedError state',
      (tester) async {
        final notifier = FakeFamilyFeedNotifier(
          const FamilyFeedError('Network error'),
        );
        await tester.pumpWidget(buildSubject(notifier));
        await tester.pump();

        expect(
          find.textContaining("Couldn't load family expenses"),
          findsOneWidget,
        );
      },
    );
  });
}
