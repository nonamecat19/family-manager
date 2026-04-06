import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:finance_tracker/features/charts/presentation/widgets/spending_bar_chart.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final testDailyTotals = [
    const DateBreakdown(date: '2026-03-01', totalCents: 5000),
    const DateBreakdown(date: '2026-03-05', totalCents: 12000),
    const DateBreakdown(date: '2026-03-15', totalCents: 8000),
  ];

  Widget buildWidget(List<DateBreakdown> dailyTotals, String month) {
    return MaterialApp(
      home: Scaffold(
        body: SpendingBarChart(dailyTotals: dailyTotals, month: month),
      ),
    );
  }

  testWidgets('renders BarChart widget', (tester) async {
    await tester.pumpWidget(buildWidget(testDailyTotals, '2026-03'));
    expect(find.byType(BarChart), findsOneWidget);
  });

  testWidgets('renders with empty data without crashing', (tester) async {
    await tester.pumpWidget(buildWidget(const [], '2026-03'));
    expect(find.byType(BarChart), findsOneWidget);
  });
}
