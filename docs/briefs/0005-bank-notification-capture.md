# 0005 — Capture bank notifications on Android and suggest finance transactions

## Goal
Once the finance app has Android notification access, it captures the notifications posted by the
banking apps the member chooses, and only those. It sends each one to a new backend service,
`services/capture`. The service stores the raw notification privately for that member and parses
it into a suggested transaction (amount, currency, direction, merchant, card/account hint, balance
when present). In the finance app the member sees an inbox of suggestions and confirms each one
into a real transaction (account and category prefilled, editable) or dismisses it. Nobody has to
type in card payments by hand any more.

## Non-goals
- No iOS capture: iOS gives no API to read other apps' notifications. On iOS the inbox screen
  shows suggestions captured on the member's Android phone, and the capture settings are hidden.
- No capture of non-bank apps unless the member adds that app to the allow-list. Capturing every
  notification is not the default (Android supports filtering by package).
- No automatic transaction creation. A suggestion always needs the member's confirmation.
- No bank API or open-banking integration, no SMS reading, no OCR.
- No sharing of raw notifications with other household members. Only confirmed transactions
  become household data.
- Not distributed through Google Play (Play restricts notification listeners). Sideloaded APK only,
  as today.
- No changes to `services/notifications` (push delivery). The new service is separate.

## Acceptance
- [ ] `libs/proto/capture/v1/capture.proto` exists and `just proto-check` passes. Its rpcs:
      `SubmitNotifications` (batch, idempotent per device notification key), `ListSuggestions`,
      `ConfirmSuggestion` (records the finance transaction id), `DismissSuggestion`,
      `ListCapturedNotifications`, `DeleteCapturedNotification`, `GetCaptureSettings` /
      `SetCaptureSettings` (allow-listed packages, enabled flag).
- [ ] `services/capture` is scaffolded like the other services (config, sqlc and migrations,
      handler, `/healthz`) and registered in `go.work`. Unit tests cover:
      - member scoping: another member, even in the same family, cannot read, confirm or delete;
      - idempotent resubmission;
      - the per-bank parsers: Monobank, PrivatBank (Privat24), PUMB, Sense/Alfa, and a generic
        "amount + currency" fallback, all against recorded sample texts in uk and en;
      - dismiss and confirm transitions;
      - retention purge.
- [ ] Migrations are additive and apply to the local compose DB.
- [ ] The Android finance app has a native notification-listener module (an Expo config plugin
      plus a headless task). A settings screen opens the system "Notification access" page, lists
      installed bank apps with known ones preselected, and stores the allow-list. Captured
      notifications are queued on the device and uploaded in batches, with retry when offline.
      Notifications from packages not on the list are dropped on the device and never uploaded.
- [ ] The finance app has an inbox screen: the suggestion list, confirm (prefilled
      `CreateTransaction`, then `ConfirmSuggestion`), and dismiss. Its label shows the pending
      count. It has en and uk strings.
- [ ] A tgemu-free e2e test drives `SubmitNotifications` with fixture texts and `ConfirmSuggestion`
      through the public listener and checks that the finance transaction exists.
- [ ] `just verify` passes, and the knowledge graph shows `service:capture`, the `capture.v1`
      contract and the finance app's edge to it.

## Affected nodes
- New: `service:capture`, `proto:libs/proto/capture/v1/capture.proto` (rpcs and messages), and
  its tables (captured notifications, suggestions, settings).
- `app:finance` (blast radius 0): new screens and the native listener module.
- `pkg:@fm/api` (3 dependents: finance, notes, recipes): a capture client and hooks. The capture
  URL is optional so notes and recipes keep working.
- `service:finance` (84 nodes): read-only use of the existing `CreateTransaction` from the app. The
  services do not call each other, so there is no contract change.
- `sdk/go`, `sdk/typescript`: regenerated with the new proto.
- `go.work`, `docker-compose.services.yml`, `justfile`.

## Known gates
- `go.work`: register `services/capture` (STOP, workspace change).
- `pnpm-lock.yaml`: the notification-listener native dependency (STOP).
- `docker-compose.services.yml` and the database init for the `capture` DB (STOP, deploy surface).
- `infra/**`: Caddy host and prod compose for the service (STOP, deploy surface). This stays a
  human step.
- Financial personal data at rest: a human reviews the storage and retention choices before the
  first deploy.

## Open questions
- Retention of raw notification text. The default is to purge raw text 90 days after the
  suggestion is confirmed or dismissed, keep pending ones until acted on, and let the member
  delete any item.
- Encryption at rest for raw text. The default is application-level AES-GCM with a service key,
  following the tasks refresh-token sealing (`services/tasks/internal/crypto`).
- Default allow-list package ids (Monobank `com.ftband.mono`, Privat24 `ua.privatbank.ap24`, PUMB,
  Sense, A-Bank, Raiffeisen). The default ships these preselected when installed. The member edits
  the list.
- Duplicates when the same purchase is also logged manually. The default puts a "possible
  duplicate" hint on a suggestion when a transaction with the same amount exists within ±1 day.
- The service name: the default is `capture`, to avoid confusion with the existing
  `notifications` push service.
