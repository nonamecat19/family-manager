import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:finance_tracker/features/family/data/models/family.dart';
import 'package:finance_tracker/features/family/data/models/family_member.dart';
import 'package:finance_tracker/features/family/domain/family_notifier.dart';
import 'package:finance_tracker/features/family/domain/family_state.dart';
import 'package:finance_tracker/features/family/presentation/family_screen.dart';
import 'package:finance_tracker/providers/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart' hide Family;
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

/// A fake [AuthNotifier] for testing.
class FakeAuthNotifier extends StateNotifier<AuthState>
    implements AuthNotifier {
  FakeAuthNotifier([AuthState initial = const AuthInitial()])
      : super(initial);

  @override
  Future<void> login(String email, String password) async {}

  @override
  Future<void> signup(String email, String password) async {}

  @override
  Future<void> logout() async {}

  @override
  Future<void> tryRestoreSession() async {}
}

void main() {
  late FakeFamilyNotifier fakeFamilyNotifier;
  late FakeAuthNotifier fakeAuthNotifier;

  Widget buildSubject(
    FakeFamilyNotifier familyNotifier,
    FakeAuthNotifier authNotifier,
  ) {
    final router = GoRouter(
      initialLocation: '/settings/family',
      routes: [
        GoRoute(
          path: '/settings/family',
          builder: (_, __) => const FamilyScreen(),
        ),
        GoRoute(
          path: '/settings/family/create',
          builder: (_, __) =>
              const Scaffold(body: Text('Create Screen')),
        ),
      ],
    );

    return ProviderScope(
      overrides: [
        familyStateProvider.overrideWith((_) => familyNotifier),
        authStateProvider.overrideWith((_) => authNotifier),
      ],
      child: MaterialApp.router(routerConfig: router),
    );
  }

  setUp(() {
    fakeAuthNotifier = FakeAuthNotifier(
      const Authenticated(userId: 'user-1', email: 'me@test.com'),
    );
  });

  group('FamilyScreen', () {
    testWidgets(
      'shows loading indicator when state is FamilyLoading',
      (tester) async {
        fakeFamilyNotifier =
            FakeFamilyNotifier(const FamilyLoading());
        await tester.pumpWidget(
          buildSubject(fakeFamilyNotifier, fakeAuthNotifier),
        );
        await tester.pump();

        expect(
          find.byType(CircularProgressIndicator),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'shows empty state with Create Family button when NoFamily',
      (tester) async {
        fakeFamilyNotifier =
            FakeFamilyNotifier(const NoFamily());
        await tester.pumpWidget(
          buildSubject(fakeFamilyNotifier, fakeAuthNotifier),
        );
        await tester.pump();

        expect(find.text('No family yet'), findsOneWidget);
        expect(find.text('Create Family'), findsOneWidget);
      },
    );

    testWidgets(
      'shows family name and members when FamilyLoaded',
      (tester) async {
        fakeFamilyNotifier = FakeFamilyNotifier(
          FamilyLoaded(
            Family(
              id: 'f1',
              name: 'The Smiths',
              adminUserId: 'user-1',
              createdAt: DateTime(2024),
            ),
            [
              FamilyMember(
                id: 'm1',
                userId: 'user-1',
                email: 'me@test.com',
                role: 'admin',
                joinedAt: DateTime(2024),
              ),
              FamilyMember(
                id: 'm2',
                userId: 'user-2',
                email: 'other@test.com',
                role: 'member',
                joinedAt: DateTime(2024),
              ),
            ],
            const [],
          ),
        );
        await tester.pumpWidget(
          buildSubject(fakeFamilyNotifier, fakeAuthNotifier),
        );
        await tester.pump();

        expect(find.text('The Smiths'), findsOneWidget);
        expect(find.text('2 members'), findsOneWidget);
        expect(find.text('me@test.com'), findsOneWidget);
        expect(find.text('other@test.com'), findsOneWidget);
      },
    );

    testWidgets(
      'admin sees Copy Invite Link button and remove member buttons',
      (tester) async {
        fakeFamilyNotifier = FakeFamilyNotifier(
          FamilyLoaded(
            Family(
              id: 'f1',
              name: 'The Smiths',
              adminUserId: 'user-1',
              createdAt: DateTime(2024),
            ),
            [
              FamilyMember(
                id: 'm1',
                userId: 'user-1',
                email: 'me@test.com',
                role: 'admin',
                joinedAt: DateTime(2024),
              ),
              FamilyMember(
                id: 'm2',
                userId: 'user-2',
                email: 'other@test.com',
                role: 'member',
                joinedAt: DateTime(2024),
              ),
            ],
            const [],
          ),
        );
        await tester.pumpWidget(
          buildSubject(fakeFamilyNotifier, fakeAuthNotifier),
        );
        await tester.pump();

        // Scroll down to reveal admin controls below the fold.
        await tester.scrollUntilVisible(
          find.text('Copy Invite Link'),
          200,
        );

        expect(find.text('Copy Invite Link'), findsOneWidget);
        expect(
          find.byIcon(Icons.remove_circle_outline),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'member does not see admin controls',
      (tester) async {
        // Auth user is user-2 (member, not admin).
        fakeAuthNotifier = FakeAuthNotifier(
          const Authenticated(
            userId: 'user-2',
            email: 'other@test.com',
          ),
        );
        fakeFamilyNotifier = FakeFamilyNotifier(
          FamilyLoaded(
            Family(
              id: 'f1',
              name: 'The Smiths',
              adminUserId: 'user-1',
              createdAt: DateTime(2024),
            ),
            [
              FamilyMember(
                id: 'm1',
                userId: 'user-1',
                email: 'me@test.com',
                role: 'admin',
                joinedAt: DateTime(2024),
              ),
              FamilyMember(
                id: 'm2',
                userId: 'user-2',
                email: 'other@test.com',
                role: 'member',
                joinedAt: DateTime(2024),
              ),
            ],
            const [],
          ),
        );
        await tester.pumpWidget(
          buildSubject(fakeFamilyNotifier, fakeAuthNotifier),
        );
        await tester.pump();

        expect(find.text('Copy Invite Link'), findsNothing);
        expect(
          find.byIcon(Icons.remove_circle_outline),
          findsNothing,
        );

        // Scroll down to reveal Leave Family button below the fold.
        await tester.scrollUntilVisible(
          find.text('Leave Family'),
          200,
        );
        expect(find.text('Leave Family'), findsOneWidget);
      },
    );

    testWidgets(
      'shows Have an invite code button in empty state',
      (tester) async {
        fakeFamilyNotifier =
            FakeFamilyNotifier(const NoFamily());
        await tester.pumpWidget(
          buildSubject(fakeFamilyNotifier, fakeAuthNotifier),
        );
        await tester.pump();

        expect(
          find.text('Have an invite code?'),
          findsOneWidget,
        );
      },
    );
  });
}
