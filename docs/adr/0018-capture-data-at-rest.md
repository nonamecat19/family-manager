# ADR 0018 — `services/capture` keeps bank notifications sealed, private to their owner, and short-lived

- Status: proposed
- Date: 2026-10-06
- Builds on: [0005](0005-auth.md) (auth), [0016](0016-notifications-service.md) (a separate
  push service)

## Context

Brief 0005 lets the Android finance app forward the notifications that the member's banking apps
post, so `services/capture` can turn them into suggested transactions. A bank notification is
personal financial data: amounts, merchants, transfer counterparties' names, card digits and
balances. Until now the repo stored no raw third-party text. Finance transactions are entered on
purpose and shared with the household; a captured notification is neither of those.

## Decision

### Ownership
Every capture row (captured notification, suggestion, prefill rule, settings, deletion tombstone)
is keyed by (family_id, user_id). Both come from the caller's token through `fmauth.Require`, never
from request fields. Every query filters by both. Other members of the same household cannot list,
confirm, dismiss or delete another member's rows. A notification becomes household data only when
its owner confirms the suggestion into a finance transaction through
`FinanceService.CreateTransaction`.
When `family.member.removed` arrives, capture deletes that member's notifications, suggestions,
prefill rules, settings and tombstones for the former family. The consumer is idempotent and uses
both ids from the event; retries cannot affect another member or family.

### Sealed in the service
The notification `title` and `text` are encrypted with AES-256-GCM under one service key,
`CAPTURE_SEAL_KEY` (base64, 32 bytes, read from the environment, never committed). Boot fails
without it. The scheme is the one `services/tasks` uses for Google refresh tokens: a version byte, a
random 12-byte nonce, and additional data bound to the domain, family id, user id, notification
row id and field name (`title` or `text`). A row or field copied to another owner, notification or
column fails to open. The service allocates the row id before sealing. The code is copied into
`services/capture/internal/crypto` rather than imported, because services never import each other.

### Stored in clear, and why
These fields are stored unencrypted, because the inbox lists, sorts and counts them, and duplicate
detection and learned prefill match on them:
- **Parsed suggestion fields:** amount, currency, direction, `merchant`, card hint (the last digits)
  and balance. On transfers, `merchant` is the counterparty, often a person's name.
- **Delivery metadata:** package name, device id, notification key, `posted_at`, `received_at`,
  `occurred_at`, parser id, and the capture settings' `allowed_packages`.
- **Outcome:** the confirmed finance transaction id.
- **`prefill_rules`:** card hint or merchant mapped to an account and category. They are learned
  from confirms and outlive the notifications they came from. Deleting a notification does not
  delete its learned rules. The member can list, delete one or clear all rules from capture
  settings. These rules have no automatic expiry.
- **Deletion tombstones:** family id, user id, device id, notification key and deletion time are
  kept in clear to reject a late retry after the owner deletes a notification. They expire one
  year after deletion.

### Retention
- **Purge.** A purge loop nulls the sealed title and text (the API reports this as `text_purged`)
  in three cases:
  - 90 days after the suggestion is confirmed or dismissed;
  - 90 days after receipt for notifications no parser understood, which have no suggestion;
  - 365 days after receipt for suggestions still pending.

  The parsed fields stay for the inbox history while the member remains in the family. Pending
  suggestions lose their raw text after 365 days even if no decision has been made.
- **Delete.** The owner can delete any captured notification at any time. The delete removes the
  row and its suggestion, and in the same transaction writes a tombstone of (family_id, user_id,
  device_id, notification_key). A late re-upload of that key is then answered `DUPLICATE` and the
  item does not come back. The tombstone holds no text.
- **Membership removal.** Removing or leaving a family deletes every capture row for that
  (family id, user id), including clear parsed data, rules, settings and tombstones.

### Device side
The app drops notifications from packages that are not on the owner's allow-list before they are
queued, so they never leave the phone. The upload queue on the phone encrypts title and text with
an AES-GCM key held in the Android Keystore, which cannot be exported. The queue is capped at 500
items and drops items older than 30 days. Items the service rejects as `DISABLED` stay queued
within those limits.
Neither the native listener nor the JS queue and uploader logs notification title, text or parsed
fields. Tests exercise error and retry paths with fixture text and assert no console output
contains it.

### Logs
Title, text and the parsed fields are never logged by the service or the app. Parser failures are
logged with the parser id and a reason code only. `rpc.Internal` and `rpc.Recover` log error and
panic values, so capture code must not wrap notification content in errors. The submit handler's
tests assert this.

### Key rotation
Rotation is not built in v1. The version byte reserves room for it: a later change can add a
keyring keyed by version, seal new rows with the newest key, and re-seal or purge old rows. Until
then, changing `CAPTURE_SEAL_KEY` makes every unpurged sealed row unreadable.

### Backups
Before the first production capture deployment, the backup job encrypts each completed backup
archive, including database dumps and deployment configuration. It does not place
`CAPTURE_SEAL_KEY` or the backup decryption key inside the archive. Both keys are stored separately
from the backup storage, with access limited to the restore operator. A restore test must verify
that the encrypted archive opens with the separately held backup key and that the separately held
capture seal key opens a captured raw notification. Failed or incomplete plaintext staging is
removed. The existing 14-run backup retention remains; deleted or purged text can still be present
in encrypted retained backups until those runs expire.

## Consequences

- **Lost key.** Losing `CAPTURE_SEAL_KEY` makes all unpurged raw text unreadable. The parsed
  suggestions stay usable.
- **Backups.** The current `infra/backup.sh` writes clear database dumps and archives `.env` and
  `secrets` beside them. The gated backup change above is required before capture's first
  production deployment; the current script does not satisfy this ADR.
- **Phone.** A rooted or compromised phone can read queued items as they are captured, before
  encryption. The Keystore protects the queue only at rest.
- **Approval.** A human must accept this ADR before the first production deploy of
  `services/capture` (brief 0005, unit hr).
