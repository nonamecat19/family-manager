import 'package:finance_tracker/core/network/connectivity_provider.dart';
import 'package:finance_tracker/shared/widgets/offline_banner.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('OfflineBanner', () {
    testWidgets('shows MaterialBanner with cloud_off icon when offline',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            connectivityProvider
                .overrideWith((_) => Stream.value(false)),
          ],
          child: const MaterialApp(
            home: Scaffold(body: OfflineBanner()),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(MaterialBanner), findsOneWidget);
      expect(find.text("You're offline"), findsOneWidget);
      expect(find.byIcon(Icons.cloud_off), findsOneWidget);
    });

    testWidgets('hides when online', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            connectivityProvider
                .overrideWith((_) => Stream.value(true)),
          ],
          child: const MaterialApp(
            home: Scaffold(body: OfflineBanner()),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(MaterialBanner), findsNothing);
      expect(find.byType(SizedBox), findsWidgets);
    });

    testWidgets('shows plural pending count when offline with 3 pending',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            connectivityProvider
                .overrideWith((_) => Stream.value(false)),
          ],
          child: const MaterialApp(
            home: Scaffold(body: OfflineBanner(pendingCount: 3)),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(
        find.text('3 expenses will sync when connected'),
        findsOneWidget,
      );
    });

    testWidgets('shows singular pending count when offline with 1 pending',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            connectivityProvider
                .overrideWith((_) => Stream.value(false)),
          ],
          child: const MaterialApp(
            home: Scaffold(body: OfflineBanner(pendingCount: 1)),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(
        find.text('1 expense will sync when connected'),
        findsOneWidget,
      );
    });

    testWidgets('does not show pending text when count is 0',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            connectivityProvider
                .overrideWith((_) => Stream.value(false)),
          ],
          child: const MaterialApp(
            home: Scaffold(body: OfflineBanner()),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text("You're offline"), findsOneWidget);
      expect(
        find.textContaining('expenses will sync'),
        findsNothing,
      );
    });
  });
}
