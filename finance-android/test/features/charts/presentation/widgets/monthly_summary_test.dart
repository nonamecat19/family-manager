import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:finance_tracker/features/charts/presentation/widgets/monthly_summary.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final testData = ChartData(
    month: '2026-03',
    totalCents: 75000,
    byCategory: const [
      CategoryBreakdown(
        categoryId: 'cat-1',
        categoryName: 'Food',
        categoryColor: '#FF7043',
        categoryIcon: 'restaurant',
        totalCents: 45000,
        count: 12,
      ),
      CategoryBreakdown(
        categoryId: 'cat-2',
        categoryName: 'Transport',
        categoryColor: '#42A5F5',
        categoryIcon: 'directions_car',
        totalCents: 30000,
        count: 8,
      ),
    ],
    byDate: const [],
  );

  Widget buildWidget(ChartData data) {
    return MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(
          child: MonthlySummary(data: data),
        ),
      ),
    );
  }

  testWidgets('shows Total Spent label', (tester) async {
    await tester.pumpWidget(buildWidget(testData));
    expect(find.text('Total Spent'), findsOneWidget);
  });

  testWidgets('shows formatted total amount', (tester) async {
    await tester.pumpWidget(buildWidget(testData));
    // 75000 cents = $750.00
    expect(find.text(r'$750.00'), findsOneWidget);
  });

  testWidgets('shows category names', (tester) async {
    await tester.pumpWidget(buildWidget(testData));
    expect(find.text('Food'), findsOneWidget);
    expect(find.text('Transport'), findsOneWidget);
  });

  testWidgets('shows expense count with pluralization', (tester) async {
    await tester.pumpWidget(buildWidget(testData));
    expect(find.text('12 expenses'), findsOneWidget);
    expect(find.text('8 expenses'), findsOneWidget);
  });

  testWidgets('shows singular for count of 1', (tester) async {
    final singleData = ChartData(
      month: '2026-03',
      totalCents: 5000,
      byCategory: const [
        CategoryBreakdown(
          categoryId: 'cat-1',
          categoryName: 'Coffee',
          categoryColor: '#8D6E63',
          categoryIcon: 'local_cafe',
          totalCents: 5000,
          count: 1,
        ),
      ],
      byDate: const [],
    );
    await tester.pumpWidget(buildWidget(singleData));
    expect(find.text('1 expense'), findsOneWidget);
  });
}
