import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:flutter/material.dart';

/// Monthly summary showing total spent and per-category breakdown.
///
/// Displays a prominent total amount followed by a list of categories
/// sorted by spending (highest first, as returned by the API).
class MonthlySummary extends StatelessWidget {
  /// Creates a [MonthlySummary].
  const MonthlySummary({required this.data, super.key});

  /// The aggregated chart data for the month.
  final ChartData data;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Total section
        Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Total Spent',
                style: theme.textTheme.bodyMedium,
              ),
              Text(
                Expense.formatCents(data.totalCents),
                style: theme.textTheme.headlineMedium,
              ),
            ],
          ),
        ),
        // Category breakdown
        ListView.separated(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          itemCount: data.byCategory.length,
          separatorBuilder: (_, __) => const Divider(indent: 56),
          itemBuilder: (context, index) {
            final cat = data.byCategory[index];
            final color = parseHexColor(cat.categoryColor);
            final iconData =
                categoryIcons[cat.categoryIcon] ?? Icons.category;
            final countLabel =
                '${cat.count} expense${cat.count == 1 ? '' : 's'}';

            return ListTile(
              leading: Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: color,
                ),
                child: Center(
                  child: Icon(iconData, size: 18, color: Colors.white),
                ),
              ),
              title: Text(cat.categoryName),
              subtitle: Text(
                countLabel,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              trailing: Text(
                Expense.formatCents(cat.totalCents),
                style: theme.textTheme.titleMedium,
              ),
            );
          },
        ),
      ],
    );
  }
}
