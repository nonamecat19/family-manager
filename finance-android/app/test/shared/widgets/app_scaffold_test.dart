import 'package:finance_tracker/core/network/connectivity_provider.dart';
import 'package:finance_tracker/features/sync/data/sync_service.dart';
import 'package:finance_tracker/shared/widgets/app_scaffold.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

Widget _wrapWithProviders(Widget child) {
  return ProviderScope(
    overrides: [
      connectivityProvider.overrideWith((_) => Stream.value(true)),
      unsyncedCountProvider.overrideWith((_) => Future.value(0)),
    ],
    child: MaterialApp(home: child),
  );
}

void main() {
  group('AppScaffold', () {
    testWidgets('renders NavigationBar', (tester) async {
      await tester.pumpWidget(
        _wrapWithProviders(
          AppScaffold(
            selectedIndex: 0,
            child: const Text('body'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(NavigationBar), findsOneWidget);
    });

    testWidgets('renders exactly 3 NavigationDestination widgets',
        (tester) async {
      await tester.pumpWidget(
        _wrapWithProviders(
          AppScaffold(
            selectedIndex: 0,
            child: const Text('body'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(NavigationDestination), findsNWidgets(3));
    });

    testWidgets('renders correct labels: History, Charts, Settings',
        (tester) async {
      await tester.pumpWidget(
        _wrapWithProviders(
          AppScaffold(
            selectedIndex: 0,
            child: const Text('body'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('History'), findsOneWidget);
      expect(find.text('Charts'), findsOneWidget);
      expect(find.text('Settings'), findsOneWidget);
    });

    testWidgets('renders FloatingActionButton with add icon',
        (tester) async {
      await tester.pumpWidget(
        _wrapWithProviders(
          AppScaffold(
            selectedIndex: 0,
            child: const Text('body'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(FloatingActionButton), findsOneWidget);
      expect(find.byIcon(Icons.add), findsOneWidget);
    });

    testWidgets('does not show offline banner when online',
        (tester) async {
      await tester.pumpWidget(
        _wrapWithProviders(
          AppScaffold(
            selectedIndex: 0,
            child: const Text('body'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(MaterialBanner), findsNothing);
    });
  });
}
