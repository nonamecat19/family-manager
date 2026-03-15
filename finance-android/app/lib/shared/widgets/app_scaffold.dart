import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// The app's main scaffold with bottom navigation and FAB.
///
/// Wraps each tab screen with a [NavigationBar] containing 3 destinations
/// (History, Charts, Settings) and a [FloatingActionButton].
class AppScaffold extends StatelessWidget {
  /// Creates an [AppScaffold].
  ///
  /// [selectedIndex] determines which tab is highlighted.
  /// [child] is the screen content for the selected tab.
  const AppScaffold({
    required this.selectedIndex,
    required this.child,
    super.key,
  });

  /// The index of the currently selected tab.
  final int selectedIndex;

  /// The screen content displayed in the body.
  final Widget child;

  static const _destinations = [
    NavigationDestination(
      icon: Icon(Icons.receipt_long),
      label: 'History',
    ),
    NavigationDestination(
      icon: Icon(Icons.pie_chart),
      label: 'Charts',
    ),
    NavigationDestination(
      icon: Icon(Icons.settings),
      label: 'Settings',
    ),
  ];

  static const _routes = ['/history', '/charts', '/settings'];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      floatingActionButton: FloatingActionButton(
        elevation: 0,
        highlightElevation: 0,
        onPressed: () => context.push('/expenses/new'),
        child: const Icon(Icons.add),
      ),
      bottomNavigationBar: NavigationBar(
        elevation: 0,
        shadowColor: Colors.transparent,
        selectedIndex: selectedIndex,
        onDestinationSelected: (index) => context.go(_routes[index]),
        destinations: _destinations,
      ),
    );
  }
}
