import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';

/// Bar chart showing daily spending totals for a month.
///
/// Renders a bar for every day of the month, filling zero for
/// days without data. Includes touch tooltip with formatted currency.
class SpendingBarChart extends StatelessWidget {
  /// Creates a [SpendingBarChart].
  const SpendingBarChart({
    required this.dailyTotals,
    required this.month,
    super.key,
  });

  /// Daily spending breakdowns from the API.
  final List<DateBreakdown> dailyTotals;

  /// Month string in "YYYY-MM" format.
  final String month;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    final parts = month.split('-');
    final year = int.parse(parts[0]);
    final monthNum = int.parse(parts[1]);
    final daysInMonth = DateTime(year, monthNum + 1, 0).day;

    // Fill all days with zero, then overlay actual data.
    final dailyCents = List<int>.filled(daysInMonth, 0);
    for (final entry in dailyTotals) {
      final day = int.parse(entry.date.split('-')[2]);
      if (day >= 1 && day <= daysInMonth) {
        dailyCents[day - 1] = entry.totalCents;
      }
    }

    return SizedBox(
      height: 200,
      child: BarChart(
        BarChartData(
          barGroups: List.generate(daysInMonth, (i) {
            return BarChartGroupData(
              x: i,
              barRods: [
                BarChartRodData(
                  toY: dailyCents[i] / 100.0,
                  color: colorScheme.primary,
                  width: daysInMonth > 28 ? 8 : 12,
                  borderRadius: const BorderRadius.vertical(
                    top: Radius.circular(4),
                  ),
                ),
              ],
            );
          }),
          gridData: FlGridData(
            drawVerticalLine: false,
            horizontalInterval: _computeInterval(dailyCents),
            getDrawingHorizontalLine: (value) => FlLine(
              color: colorScheme.outlineVariant,
              strokeWidth: 0.5,
            ),
          ),
          borderData: FlBorderData(show: false),
          titlesData: FlTitlesData(
            topTitles: const AxisTitles(),
            rightTitles: const AxisTitles(),
            bottomTitles: AxisTitles(
              sideTitles: SideTitles(
                showTitles: true,
                reservedSize: 24,
                getTitlesWidget: (value, meta) {
                  final day = value.toInt() + 1;
                  // Show day 1, every 5th day, and last day.
                  if (day == 1 ||
                      day % 5 == 1 ||
                      day == daysInMonth) {
                    return SideTitleWidget(
                      meta: meta,
                      child: Text(
                        '$day',
                        style: theme.textTheme.bodySmall
                            ?.copyWith(fontSize: 10),
                      ),
                    );
                  }
                  return const SizedBox.shrink();
                },
              ),
            ),
            leftTitles: AxisTitles(
              sideTitles: SideTitles(
                showTitles: true,
                reservedSize: 48,
                getTitlesWidget: (value, meta) {
                  return SideTitleWidget(
                    meta: meta,
                    child: Text(
                      _formatAxisLabel(value),
                      style: theme.textTheme.bodySmall
                          ?.copyWith(fontSize: 10),
                    ),
                  );
                },
              ),
            ),
          ),
          barTouchData: BarTouchData(
            touchTooltipData: BarTouchTooltipData(
              getTooltipColor: (_) => colorScheme.inverseSurface,
              getTooltipItem: (group, groupIndex, rod, rodIndex) {
                final cents = dailyCents[group.x];
                return BarTooltipItem(
                  Expense.formatCents(cents),
                  TextStyle(color: colorScheme.onInverseSurface),
                );
              },
            ),
          ),
        ),
      ),
    );
  }

  /// Computes a reasonable horizontal grid interval.
  double _computeInterval(List<int> dailyCents) {
    final maxCents = dailyCents.fold(0, (a, b) => a > b ? a : b);
    final maxDollars = maxCents / 100.0;
    if (maxDollars <= 50) return 10;
    if (maxDollars <= 200) return 50;
    if (maxDollars <= 1000) return 200;
    return 500;
  }

  /// Formats axis label: 0 -> "\$0", <1000 -> "\$X00", >=1000 -> "\$X.XK".
  String _formatAxisLabel(double value) {
    if (value == 0) return r'$0';
    if (value >= 1000) {
      final k = value / 1000;
      return k == k.roundToDouble()
          ? '\$${k.toInt()}K'
          : '\$${k.toStringAsFixed(1)}K';
    }
    return '\$${value.toInt()}';
  }
}
