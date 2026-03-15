---
phase: 02-authentication
plan: 02
subsystem: auth
tags: [flutter, riverpod, dio, go_router, flutter_secure_storage, jwt, widget-tests]

# Dependency graph
requires:
  - phase: 02-authentication
    plan: 01
    provides: Go backend auth API (signup, login, refresh, logout endpoints)
  - phase: 01-foundation
    provides: Flutter app shell, Riverpod, go_router, Dio client, flat design theme
provides:
  - Welcome screen with Log In and Sign Up navigation
  - Login screen with email/password form and inline field errors
  - Signup screen with email/password form and inline field errors
  - AuthNotifier (Riverpod StateNotifier) managing auth state lifecycle
  - SecureStorageService for persistent JWT token storage
  - AuthInterceptor (QueuedInterceptor) for token injection and 401 refresh
  - GoRouter auth redirect guard (unauthenticated -> welcome, authenticated -> history)
  - Logout from settings screen
  - Session restoration on app restart
affects: [03-expense-tracking]

# Tech tracking
tech-stack:
  added: [flutter_secure_storage 10.0.0, mocktail 1.0.0]
  patterns: [ConsumerStatefulWidget for form screens, StateNotifier for auth state, QueuedInterceptor for token management, Provider-based GoRouter with auth redirect]

key-files:
  created:
    - app/lib/core/storage/secure_storage.dart
    - app/lib/core/network/auth_interceptor.dart
    - app/lib/features/auth/data/models/auth_response.dart
    - app/lib/features/auth/data/auth_repository.dart
    - app/lib/features/auth/domain/auth_state.dart
    - app/lib/providers/auth_provider.dart
    - app/lib/features/auth/presentation/welcome_screen.dart
    - app/lib/features/auth/presentation/login_screen.dart
    - app/lib/features/auth/presentation/signup_screen.dart
    - app/test/core/network/auth_interceptor_test.dart
    - app/test/features/auth/presentation/welcome_screen_test.dart
    - app/test/features/auth/presentation/login_screen_test.dart
    - app/test/features/auth/presentation/signup_screen_test.dart
    - app/test/core/router/app_router_test.dart
  modified:
    - app/lib/core/network/api_client.dart
    - app/lib/core/router/app_router.dart
    - app/lib/app.dart
    - app/lib/main.dart
    - app/lib/features/settings/presentation/settings_screen.dart
    - app/pubspec.yaml

key-decisions:
  - "Separate refreshDio for token refresh calls to avoid recursive interceptor invocation"
  - "AuthRepository uses plain Dio (not dioProvider) for auth calls since those are pre-authentication"
  - "FakeAuthNotifier pattern in tests for clean Riverpod provider overrides"
  - "Provider-based GoRouter that rebuilds on auth state change for reactive redirect"

patterns-established:
  - "ConsumerStatefulWidget with form validation: autovalidate toggles on first submit"
  - "FakeAuthNotifier for widget test mocking via authStateProvider.overrideWith"
  - "Field-level error propagation: server errors parsed in repository, mapped to fieldErrors map, displayed inline under form fields"

requirements-completed: [AUTH-01, AUTH-02]

# Metrics
duration: 3min
completed: 2026-03-15
---

# Phase 2 Plan 2: Flutter Authentication UI Summary

**Riverpod auth state with welcome/login/signup screens, QueuedInterceptor token refresh, GoRouter redirect guard, and 19 passing tests**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T14:24:01Z
- **Completed:** 2026-03-15T14:27:00Z
- **Tasks:** 3/3 (Task 3 checkpoint approved)
- **Files modified:** 20

## Accomplishments
- Complete auth UI: welcome screen with branding, login and signup screens with real-time form validation and inline error display
- Auth data layer: repository with server error parsing to field-specific messages, sealed AuthState hierarchy, SecureStorageService token persistence
- QueuedInterceptor handles automatic token injection on requests and transparent 401 refresh with separate Dio to prevent recursion
- GoRouter provider with auth-aware redirect -- unauthenticated users see welcome screen, authenticated users bypass auth routes
- Session persistence across app restart via tryRestoreSession with token refresh
- Logout from settings clears tokens and returns to welcome screen
- 19 tests passing: 2 interceptor unit tests, 3 welcome navigation tests, 4 login validation tests, 4 signup validation tests, 2 router redirect tests, 4 scaffold tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Auth data layer, state management, and interceptor** - `da3cdf8` (feat)
2. **Task 2: Auth screens, router guards, logout, and widget tests** - `4270f70` (feat)
3. **Task 3: Verify complete authentication flow end-to-end** - checkpoint:human-verify APPROVED

## Files Created/Modified
- `app/lib/core/storage/secure_storage.dart` - FlutterSecureStorage wrapper for token and user persistence
- `app/lib/core/network/auth_interceptor.dart` - QueuedInterceptor for Bearer token injection and 401 refresh
- `app/lib/core/network/api_client.dart` - Dio provider with auth interceptor and separate refresh Dio
- `app/lib/features/auth/data/models/auth_response.dart` - AuthResponse model with fromJson factory
- `app/lib/features/auth/data/auth_repository.dart` - AuthRepository with login/signup/refresh/logout and error parsing
- `app/lib/features/auth/domain/auth_state.dart` - Sealed AuthState: AuthInitial, AuthLoading, Authenticated, Unauthenticated, AuthError
- `app/lib/providers/auth_provider.dart` - AuthNotifier StateNotifier with login, signup, logout, tryRestoreSession
- `app/lib/features/auth/presentation/welcome_screen.dart` - Welcome screen with app branding and navigation buttons
- `app/lib/features/auth/presentation/login_screen.dart` - Login form with email/password validation and inline errors
- `app/lib/features/auth/presentation/signup_screen.dart` - Signup form with 8-72 char password validation and inline errors
- `app/lib/core/router/app_router.dart` - GoRouter provider with auth-aware redirect logic
- `app/lib/app.dart` - Root widget with loading splash during session restore
- `app/lib/main.dart` - Entry point with ProviderContainer and session restoration
- `app/lib/features/settings/presentation/settings_screen.dart` - Settings with Log Out button
- `app/test/core/network/auth_interceptor_test.dart` - Interceptor token header tests
- `app/test/features/auth/presentation/welcome_screen_test.dart` - Welcome screen navigation tests
- `app/test/features/auth/presentation/login_screen_test.dart` - Login validation tests
- `app/test/features/auth/presentation/signup_screen_test.dart` - Signup validation tests
- `app/test/core/router/app_router_test.dart` - Router redirect tests

## Decisions Made
- Separate refreshDio instance for token refresh calls to avoid recursive interceptor invocation (per research anti-pattern)
- AuthRepository uses plain Dio (not the dioProvider with auth interceptor) since auth endpoints don't need Bearer tokens
- FakeAuthNotifier pattern for widget testing -- implements AuthNotifier with no-op methods, used via authStateProvider.overrideWith
- Provider-based GoRouter that watches authStateProvider and rebuilds on state change for reactive redirect
- Autovalidate mode toggles on first form submission attempt (not immediately on field interaction)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Checkpoint Verification

**Task 3: checkpoint:human-verify** -- APPROVED
- User confirmed all 10 verification steps pass (welcome screen, signup, login, validation errors, session persistence, logout)
- End-to-end auth flow verified working against Go backend

## Next Phase Readiness
- Flutter auth UI complete and verified end-to-end
- Auth state management wired for all future authenticated features
- AuthInterceptor will transparently handle token refresh for expense tracking API calls in Phase 3
- GoRouter redirect guard protects all authenticated routes

## Self-Check: PASSED

---
*Phase: 02-authentication*
*Completed: 2026-03-15*
