import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:finance_tracker/features/charts/presentation/widgets/spending_pie_chart.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final testCategories = [
    const CategoryBreakdown(
      categoryId: 'cat-1',
      categoryName: 'Food',
      categoryColor: '#FF7043',
      categoryIcon: 'restaurant',
      totalCents: 45000,
      count: 12,
    ),
    const CategoryBreakdown(
      categoryId: 'cat-2',
      categoryName: 'Transport',
      categoryColor: '#42A5F5',
      categoryIcon: 'directions_car',
      totalCents: 30000,
      count: 8,
    ),
  ];

  Widget buildWidget(List<CategoryBreakdown> categories) {
    return MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(
          child: SpendingPieChart(categories: categories),
        ),
      ),
    );
  }

  testWidgets('renders PieChart widget', (tester) async {
    await tester.pumpWidget(buildWidget(testCategories));
    expect(find.byType(PieChart), findsOneWidget);
  });

  testWidgets('shows category names in legend', (tester) async {
    await tester.pumpWidget(buildWidget(testCategories));
    expect(find.text('Food'), findsOneWidget);
    expect(find.text('Transport'), findsOneWidget);
  });

  testWidgets('shows percentage in legend', (tester) async {
    await tester.pumpWidget(buildWidget(testCategories));
    // Food = 45000/75000 = 60%, Transport = 30000/75000 = 40%
    expect(find.text('60%'), findsOneWidget);
    expect(find.text('40%'), findsOneWidget);
  });
}
