# 0004 — Interactive home-screen widgets for every app (Android and iOS)

## Goal
Every mobile app (finance, notes, recipes, tasks) offers real home-screen widgets on Android and
iOS. A widget shows live family data without opening the app, and lets the user act from the home
screen. Finance widgets render the six types that already exist as in-app previews and server data
(`WidgetType`: quick add, month, category, budgets and family, recent transactions, accounts),
using `FinanceService.GetWidgetData`. Quick add logs a template or an amount from the widget. Notes
gets "recent / pinned notes" and "new note". Recipes gets "today's meal plan" and "shopping basket",
where an item can be ticked off. Tasks gets "today and overdue", where a task can be marked done,
and "upcoming birthdays". Tapping a widget opens the matching screen through a deep link. Widgets
refresh after any write in the app, on a periodic schedule, and after a widget action.

## Non-goals
- No new backend widget model for notes, recipes or tasks. Those widgets read the existing list
  RPCs. Only finance keeps its server-side `widget_instances` configuration.
- No lock-screen widgets, Live Activities, watch complications or Android Glance-only features.
- No widget content while signed out, other than a "Sign in" tile.
- No offline write queue: if an action from a widget fails, the widget shows an error state and the
  app retries nothing.
- No change to auth flows or token lifetimes. Widgets reuse the app's stored session.

## Acceptance
- [ ] A shared package (`packages/widgets` or an extension of `@fm/ui`) gives each app one way to
      declare widgets, publish a data snapshot, and handle widget actions on both platforms.
- [ ] Android: each app registers its widgets (an Expo config plugin plus an AppWidgetProvider,
      for example `react-native-android-widget`). The widgets render on the emulator, deep-link on
      tap, and run their actions. The Maestro flow under `.maestro/flows/widgets/` covers adding
      one widget per app and one action.
- [ ] iOS: each app has a WidgetKit extension (an Expo config plugin for the target, an App Group
      for the shared snapshot and the session). `expo prebuild -p ios` succeeds and the Xcode
      project compiles in CI (`just check-ios-widgets` or an equivalent). Interactive actions use
      App Intents (iOS 17+). On older iOS the widget is tap-to-open only.
- [ ] Widget actions call the services with the user's existing session, and only the same
      RPCs the app uses: `LogTemplate` / `CreateTransaction`, `CompleteTask`, the basket update,
      and `CreateNote`. A failed or expired session shows "Open app to sign in".
- [ ] Data is written to the widget snapshot after app writes and on a periodic refresh
      (Android WorkManager / iOS timeline, at most every 15 minutes). The snapshot holds only what
      the widget shows.
- [ ] `just check-ts` passes; each app still builds a release APK with
      `EXPO_PUBLIC_API_ENV=emulator`; `just verify` is green.
- [ ] Widget strings are translated (en, uk) in each app's dictionary.

## Affected nodes
- `app:finance`, `app:notes`, `app:recipes` (blast radius 0 each) and `app:tasks` (from brief
  0003, not built yet).
- `pkg:@fm/api` (3 dependents: finance, notes, recipes): hooks that a widget's background task can
  call outside React.
- `pkg:@fm/ui` (3 dependents), only if the shared widget kit lives there.
- `service:finance` (84 nodes): read-only use of `GetWidgetData`, `LogTemplate` and
  `CreateTransaction`. No contract change.
- `pkg:@fm/auth`: reading the stored session from a widget process (App Group or shared
  SharedPreferences). The read path only. No change to auth logic.

## Known gates
- `pnpm-lock.yaml`: new native dependencies (the Android widget library, the iOS target plugin).
  STOP, workspace-wide dependency change.
- `packages/config` if the Expo/TypeScript base config needs a plugin entry. STOP, repo-wide.
- Moving the session into shared storage that the widget process can read (an iOS App Group
  keychain/UserDefaults, a shared Android store) touches how `@fm/auth` stores tokens. This is
  security-sensitive, so it gets a human review even though `packages/auth` is not in the STOP
  column.
- `.github/workflows/**` if the iOS compile check runs in CI. STOP, CI surface.
- Apple Developer team ID / App Group identifiers are signing configuration. Human-provided
  credential values.

## Open questions
- iOS cannot be verified on the Android emulator flow. The default is that the iOS acceptance is
  "prebuild and compile" only, with no simulator UI test.
- iOS interactive widgets need iOS 17. The default is App Intents on 17+ and tap-to-open below it.
- Which finance widget sizes map to the Android and iOS families? The default maps `4x1`/`4x2` to
  medium, `2x2`/`2x1` to small and `4x3` to large, and drops sizes a platform cannot render.
- Should a widget action be allowed while the phone is locked? The default is no on iOS (actions
  require unlock), and Android behaves as the launcher allows.
