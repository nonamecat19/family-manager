import 'package:finance_tracker/features/family/data/models/family_summary.dart';
import 'package:finance_tracker/features/family/domain/family_summary_notifier.dart';
import 'package:finance_tracker/features/family/presentation/family_summary_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Widget buildSubject(FakeFamilySummaryNotifier notifier) {
    return ProviderScope(
      overrides: [
        familySummaryStateProvider.overrideWith((_) => notifier),
      ],
      child: const MaterialApp(home: FamilySummaryScreen()),
    );
  }

  group('FamilySummaryScreen', () {
    testWidgets(
      'shows loading indicator for FamilySummaryLoading state',
      (tester) async {
        final notifier =
            FakeFamilySummaryNotifier(const FamilySummaryLoading());
        await tester.pumpWidget(buildSubject(notifier));
        await tester.pump();

        expect(
          find.byType(CircularProgressIndicator),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'shows per-person and per-category totals for loaded state',
      (tester) async {
        const testSummary = FamilySummary(
          totalCents: 5000,
          byPerson: [
            const MemberTotal(
              userId: 'u1',
              userEmail: 'alice@test.com',
              totalCents: 3000,
              expenseCount: 2,
            ),
            const MemberTotal(
              userId: 'u2',
              userEmail: 'bob@test.com',
              totalCents: 2000,
              expenseCount: 1,
            ),
          ],
          byCategory: [
            const FamilyCategoryTotal(
              categoryId: 'c1',
              categoryName: 'Food',
              categoryColor: '#FF7043',
              categoryIcon: 'restaurant',
              totalCents: 3500,
              expenseCount: 2,
            ),
            const FamilyCategoryTotal(
              categoryId: 'c2',
              categoryName: 'Transport',
              categoryColor: '#42A5F5',
              categoryIcon: 'directions_car',
              totalCents: 1500,
              expenseCount: 1,
            ),
          ],
        );
        final notifier = FakeFamilySummaryNotifier(
          FamilySummaryLoaded(testSummary),
        );
        await tester.pumpWidget(buildSubject(notifier));
        await tester.pump();

        expect(find.text(r'$50.00'), findsOneWidget);
        expect(find.text('By Person'), findsOneWidget);
        expect(find.text('By Category'), findsOneWidget);
        expect(find.text('alice@test.com'), findsOneWidget);
        expect(find.text('bob@test.com'), findsOneWidget);
        expect(find.text('Food'), findsOneWidget);
        expect(find.text('Transport'), findsOneWidget);
      },
    );

    testWidgets(
      'shows empty state when no spending data',
      (tester) async {
        final notifier = FakeFamilySummaryNotifier(
          const FamilySummaryLoaded(
            FamilySummary(
              totalCents: 0,
              byPerson: [],
              byCategory: [],
            ),
          ),
        );
        await tester.pumpWidget(buildSubject(notifier));
        await tester.pump();

        expect(find.text('No spending data'), findsOneWidget);
      },
    );
  });
}
