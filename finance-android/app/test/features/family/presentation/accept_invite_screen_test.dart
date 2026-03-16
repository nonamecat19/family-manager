import 'dart:async';

import 'package:finance_tracker/features/family/data/family_repository.dart';
import 'package:finance_tracker/features/family/domain/family_notifier.dart';
import 'package:finance_tracker/features/family/domain/family_state.dart';
import 'package:finance_tracker/features/family/presentation/accept_invite_screen.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart' hide Family;
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:mocktail/mocktail.dart';

class MockFamilyRepository extends Mock implements FamilyRepository {}

void main() {
  late FakeFamilyNotifier fakeFamilyNotifier;
  late MockFamilyRepository mockRepository;

  setUp(() {
    fakeFamilyNotifier = FakeFamilyNotifier(const NoFamily());
    mockRepository = MockFamilyRepository();
  });

  Widget buildSubject({
    String token = 'test-token',
  }) {
    final router = GoRouter(
      initialLocation: '/invite/$token',
      routes: [
        GoRoute(
          path: '/invite/:token',
          builder: (_, state) => AcceptInviteScreen(
            token: state.pathParameters['token']!,
          ),
        ),
        GoRoute(
          path: '/settings/family',
          builder: (_, __) =>
              const Scaffold(body: Text('Family Screen')),
        ),
      ],
    );

    return ProviderScope(
      overrides: [
        familyStateProvider.overrideWith((_) => fakeFamilyNotifier),
        familyRepositoryProvider
            .overrideWithValue(mockRepository),
      ],
      child: MaterialApp.router(routerConfig: router),
    );
  }

  group('AcceptInviteScreen', () {
    testWidgets(
      'shows loading indicator initially',
      (tester) async {
        final completer = Completer<Map<String, dynamic>>();
        when(() => mockRepository.getInvitationInfo(any()))
            .thenAnswer((_) => completer.future);

        await tester.pumpWidget(buildSubject());
        await tester.pump();

        expect(
          find.byType(CircularProgressIndicator),
          findsOneWidget,
        );

        // Complete the future to avoid pending timer issues.
        completer.complete({'family_name': 'Test'});
        await tester.pumpAndSettle();
      },
    );

    testWidgets(
      'shows invitation preview with family name and '
      'Join Family button',
      (tester) async {
        when(() => mockRepository.getInvitationInfo(any()))
            .thenAnswer(
          (_) async => <String, dynamic>{
            'family_name': 'The Johnsons',
          },
        );

        await tester.pumpWidget(buildSubject());
        await tester.pumpAndSettle();

        expect(find.text("You're invited!"), findsOneWidget);
        expect(find.text('Join The Johnsons'), findsOneWidget);
        expect(
          find.widgetWithText(FilledButton, 'Join Family'),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'shows error state for invalid/expired invitation',
      (tester) async {
        when(() => mockRepository.getInvitationInfo(any()))
            .thenThrow(
          DioException(
            requestOptions:
                RequestOptions(path: '/invitations/bad'),
            response: Response(
              requestOptions:
                  RequestOptions(path: '/invitations/bad'),
              statusCode: 404,
            ),
          ),
        );

        await tester.pumpWidget(buildSubject());
        await tester.pumpAndSettle();

        expect(find.text('Invalid Invitation'), findsOneWidget);
        expect(find.text('Go Back'), findsOneWidget);
      },
    );
  });
}
