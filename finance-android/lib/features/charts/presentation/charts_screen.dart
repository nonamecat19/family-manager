import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:finance_tracker/features/charts/presentation/widgets/monthly_summary.dart';
import 'package:finance_tracker/features/charts/presentation/widgets/spending_bar_chart.dart';
import 'package:finance_tracker/features/charts/presentation/widgets/spending_pie_chart.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:finance_tracker/providers/chart_provider.dart';
import 'package:finance_tracker/providers/expense_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

/// Charts tab screen showing spending visualizations.
///
/// Displays a pie chart (category breakdown), bar chart (daily totals),
/// and monthly summary list. Users can navigate between months using
/// arrow buttons. Charts automatically reload when expenses change
/// on other tabs.
class ChartsScreen extends ConsumerStatefulWidget {
  /// Creates a [ChartsScreen].
  const ChartsScreen({super.key});

  @override
  ConsumerState<ChartsScreen> createState() => _ChartsScreenState();
}

class _ChartsScreenState extends ConsumerState<ChartsScreen> {
  late DateTime _selectedMonth;

  @override
  void initState() {
    super.initState();
    final now = DateTime.now();
    _selectedMonth = DateTime(now.year, now.month);
    // Load data after the first frame so ref is available.
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadData());
  }

  void _loadData() {
    final formatted = DateFormat('yyyy-MM').format(_selectedMonth);
    ref.read(chartStateProvider.notifier).loadSummary(month: formatted);
  }

  void _previousMonth() {
    setState(() {
      _selectedMonth = DateTime(
        _selectedMonth.year,
        _selectedMonth.month - 1,
      );
    });
    _loadData();
  }

  void _nextMonth() {
    setState(() {
      _selectedMonth = DateTime(
        _selectedMonth.year,
        _selectedMonth.month + 1,
      );
    });
    _loadData();
  }

  bool get _isCurrentMonth {
    final now = DateTime.now();
    return _selectedMonth.year == now.year &&
        _selectedMonth.month == now.month;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final chartState = ref.watch(chartStateProvider);

    // Reload charts when expenses change on other tabs.
    ref.listen<ExpenseState>(expenseStateProvider, (previous, next) {
      if (next is ExpenseLoaded) {
        _loadData();
      }
    });

    return Column(
      children: [
        // Month selector
        Padding(
          padding: const EdgeInsets.only(
            left: 16,
            right: 16,
            top: 8,
            bottom: 16,
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              IconButton(
                icon: Icon(
                  Icons.chevron_left,
                  color: colorScheme.primary,
                  size: 24,
                ),
                onPressed: _previousMonth,
              ),
              Text(
                DateFormat.yMMMM().format(_selectedMonth),
                style: theme.textTheme.titleMedium,
              ),
              IconButton(
                icon: Icon(
                  Icons.chevron_right,
                  color: _isCurrentMonth
                      ? colorScheme.onSurface.withAlpha(97)
                      : colorScheme.primary,
                  size: 24,
                ),
                onPressed: _isCurrentMonth ? null : _nextMonth,
              ),
            ],
          ),
        ),
        // Chart content
        Expanded(
          child: _buildContent(chartState, theme, colorScheme),
        ),
      ],
    );
  }

  Widget _buildContent(
    ChartState chartState,
    ThemeData theme,
    ColorScheme colorScheme,
  ) {
    return switch (chartState) {
      ChartInitial() || ChartLoading() => const Center(
          child: CircularProgressIndicator(),
        ),
      ChartError() => RefreshIndicator(
          onRefresh: () async => _loadData(),
          child: SingleChildScrollView(
            physics: const AlwaysScrollableScrollPhysics(),
            child: SizedBox(
              height: MediaQuery.of(context).size.height * 0.5,
              child: Center(
                child: Text(
                  "Couldn't load spending summary. Pull down to try again.",
                  style: theme.textTheme.bodyMedium,
                  textAlign: TextAlign.center,
                ),
              ),
            ),
          ),
        ),
      ChartLoaded(:final data) => data.byCategory.isEmpty
          ? Center(
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 32),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(
                      Icons.pie_chart_outline,
                      size: 48,
                      color: colorScheme.onSurfaceVariant,
                    ),
                    const SizedBox(height: 16),
                    Text(
                      'No spending data',
                      style: theme.textTheme.titleMedium,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'Start logging expenses to see your spending '
                      'breakdown here.',
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                      textAlign: TextAlign.center,
                    ),
                  ],
                ),
              ),
            )
          : RefreshIndicator(
              onRefresh: () async => _loadData(),
              child: SingleChildScrollView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Column(
                  children: [
                    SpendingPieChart(categories: data.byCategory),
                    const SizedBox(height: 24),
                    SpendingBarChart(
                      dailyTotals: data.byDate,
                      month: data.month,
                    ),
                    const SizedBox(height: 24),
                    MonthlySummary(data: data),
                    const SizedBox(height: 24),
                  ],
                ),
              ),
            ),
    };
  }
}
