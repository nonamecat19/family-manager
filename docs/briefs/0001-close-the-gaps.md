# Brief 0001 — Close the known gaps

Every feature the docs and code promise but do not deliver, built in one run.

1. Notes sharing: `ShareNote`, `ShareNotebook`, `Unshare`, `ListShares`, `ListSharedWithMe` in
   `services/notes` (Java), backed by a Flyway migration.
2. Notes image upload: `UploadNoteImage` stored in MinIO per ADR 0007.
3. A "Connected accounts" screen in every app, on the existing `ListIdentities` / `Unlink`.
4. Device-code login in `services/auth`: a client without a browser asks for a code, a signed-in
   user approves it from an app, the client polls and receives tokens.
5. Log in with Telegram on mobile, built on the same pending-login grant: the app opens the bot
   with a login code, the bot approves it for the Telegram identity already linked.
6. A terminal client (`apps/tui`, Go) that signs in with the device-code flow.
7. `services/notifications`: registers Expo push tokens per user and device, consumes domain
   events from JetStream and sends push notifications; apps register their push token.
8. Prometheus metrics for every Go service (`/metrics`, RPC counters and latency) and the Java
   notes service (actuator), with a Prometheus container in compose.
9. Move `postgres/init/` to `infra/postgres/`; delete the empty `services/notes-java/`; bring
   `docs/architecture.md` up to date.

Delivery: commits on master, pushed. Migrations applied to the local database only.
