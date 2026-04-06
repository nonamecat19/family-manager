import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';

/// Pie chart showing spending distribution by category.
///
/// Displays a colored pie chart with touch interaction (sections
/// expand on tap) and a legend below showing category names and
/// percentages.
class SpendingPieChart extends StatefulWidget {
  /// Creates a [SpendingPieChart].
  const SpendingPieChart({required this.categories, super.key});

  /// Category spending breakdowns to visualize.
  final List<CategoryBreakdown> categories;

  @override
  State<SpendingPieChart> createState() => _SpendingPieChartState();
}

class _SpendingPieChartState extends State<SpendingPieChart> {
  int _touchedIndex = -1;

  int get _totalCents =>
      widget.categories.fold(0, (sum, cat) => sum + cat.totalCents);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final totalCents = _totalCents;

    return Column(
      children: [
        SizedBox(
          height: 220,
          child: PieChart(
            PieChartData(
              pieTouchData: PieTouchData(
                touchCallback: (event, response) {
                  setState(() {
                    if (!event.isInterestedForInteractions ||
                        response == null ||
                        response.touchedSection == null) {
                      _touchedIndex = -1;
                      return;
                    }
                    _touchedIndex =
                        response.touchedSection!.touchedSectionIndex;
                  });
                },
              ),
              sections: _buildSections(),
              centerSpaceRadius: 40,
              sectionsSpace: 2,
            ),
            duration: const Duration(milliseconds: 300),
          ),
        ),
        const SizedBox(height: 24),
        Wrap(
          spacing: 8,
          runSpacing: 4,
          children: widget.categories.map((cat) {
            final percent = totalCents > 0
                ? cat.totalCents / totalCents * 100
                : 0.0;
            final percentText =
                percent >= 1 ? '${percent.round()}%' : '< 1%';

            return Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 12,
                  height: 12,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: parseHexColor(cat.categoryColor),
                  ),
                ),
                const SizedBox(width: 8),
                Text(
                  cat.categoryName,
                  style: theme.textTheme.bodyMedium,
                ),
                const SizedBox(width: 4),
                Text(
                  percentText,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            );
          }).toList(),
        ),
      ],
    );
  }

  List<PieChartSectionData> _buildSections() {
    return widget.categories.asMap().entries.map((entry) {
      final index = entry.key;
      final cat = entry.value;
      final isTouched = index == _touchedIndex;

      return PieChartSectionData(
        value: cat.totalCents.toDouble(),
        color: parseHexColor(cat.categoryColor),
        title: cat.categoryName,
        radius: isTouched ? 110 : 100,
        titleStyle: const TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.bold,
          color: Colors.white,
        ),
      );
    }).toList();
  }
}
