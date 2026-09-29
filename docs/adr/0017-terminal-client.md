# ADR 0017 — A Go + Bubble Tea terminal client, `fm`, signs in with a device code

- Status: accepted
- Date: 2026-10-04
- Builds on: [0005](0005-auth.md) (auth), [0013](0013-linked-identities.md) (identities),
  [0014](0014-device-and-telegram-login.md) (device-code login)

## Context

ADR 0013 and ADR 0014 named a terminal client as the first consumer of device-code login, and
the auth side shipped: `StartDeviceLogin` / `PollDeviceLogin` are public, and every Expo app has
Settings → Approve a device. Nothing used them. Someone at a shell had no way into their notes
without a phone in hand, and the device-code flow had no client that exercised it end to end.

Every client so far is an Expo app in `apps/`, built by Turborepo, talking to services through
`packages/api`. A terminal program fits none of that: no React Native, no Metro, no secure store.

## Decision

**`apps/tui` is a Go module, `github.com/nnc/family-manager/apps/tui`, building one binary, `fm`.**
It is registered in `go.work` and is built, vetted and tested by `just check-go` like every
other module. It lives in `apps/` because it is a client a person runs, not a library or a
service; it has no `package.json`, so Turborepo and pnpm never see it.

- **Transport.** The generated Connect clients in `sdk/go` (`authv1connect`, `notesv1connect`),
  Connect protocol with JSON over HTTP/1.1, the same wire format the Expo apps use. Base URLs come
  from `--auth-url` / `--notes-url`, then `FM_AUTH_URL` / `FM_NOTES_URL`, then a preset picked by
  `--env` / `FM_ENV`: `production` uses the Caddy hosts (`AUTH_HOST`, `NOTES_HOST`), `local`
  uses the ports the Expo apps use in development.
- **Sign-in is device-code only.** `fm login` calls `StartDeviceLogin(kind=DEVICE)`, prints the
  user code and tells the person to approve it in any family-manager app under Settings →
  Approve a device, then polls `PollDeviceLogin` at `interval_seconds`, lengthening the interval
  on `SLOW_DOWN` to the larger of the server's value and the old interval plus five seconds, and
  stops on `APPROVED`, `DENIED`, `EXPIRED` or the grant's local expiry. There is no password
  prompt: a password typed into a terminal ends up in shell history and scrollback, and the
  device flow already exists.
- **Credentials** are the access token, refresh token and expiry, written as JSON to
  `os.UserConfigDir()/family-manager/credentials.json` — file `0600`, directory `0700`, replaced
  atomically through a temp file and rename. No OS keychain: it would add a cgo or D-Bus
  dependency per platform for a token that is already short-lived and revocable.
- **Refresh is transparent.** Every authenticated call goes through one interceptor that reads
  the stored pair and, within 30 seconds of expiry, calls `Refresh` and saves the rotated pair
  before sending. A refresh rejected with `Unauthenticated` deletes the file and the client is
  logged out; any other refresh failure keeps the file.
- **`fm whoami`** decodes the access token's claims without verifying the signature. The client
  never makes authorization decisions from them; it only prints `sub`, `family_id` and `email`.
  Verification stays where ADR 0005 put it, in the services.
- **`fm logout`** calls `Logout` with the refresh token, then deletes the file even if the call
  failed, and says so.
- **`fm` with no arguments** opens a Bubble Tea UI: the device-code screen when logged out, the
  notes list from `ListNotes` otherwise, and a read-only view of a note's text blocks from
  `GetNote`. Writing is out of scope.
- **Bubble Tea v2 and lipgloss v2** (`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`). The
  workspace already resolves lipgloss v2 and `charmbracelet/x/ansi` v0.11 through `minio-go`,
  and `go.work` gives every module one version of each dependency; Bubble Tea v1 with lipgloss
  v1 does not compile against that `x/ansi`.

## Alternatives

- **A TypeScript CLI reusing `packages/api` and `packages/auth`** (Ink for the UI). Shares the
  query and auth code, but `packages/auth` is built on expo-secure-store and React state, and
  the binary would need Node on the user's machine. A Go binary is one file and reuses the
  generated Go clients the services already use.
- **Email and password in the terminal.** Puts the password in a terminal and duplicates a
  login path the device flow made unnecessary.
- **A loopback-redirect browser login.** Needs a browser on the same machine, which is exactly
  the case a TUI over SSH does not have.
- **`apps/tui` under `tools/` or a new top-level `clients/`.** `tools/` is repo tooling, not
  something a family member runs; a new top-level directory for one module is layout churn.

## Consequences

- `apps/` now holds one Go module next to the Expo apps. Anything that assumes every `apps/*`
  has a `package.json` must skip it; `go.work` lists it.
- Credentials sit in a plain file guarded by permissions. Anyone who can read the user's home
  directory can use the session until it is revoked with `fm logout` or from another client.
- `fm` is the first client to call `StartDeviceLogin` with kind `DEVICE`; the follow-up in ADR
  0014 about rate-limiting that public RPC per source address applies to it.
- Writing notes, other domains (finance, recipes) and a release pipeline for the binary are
  follow-ups.
