# ADR 0009 — Tauri v2 wrapping the Expo web export for the notes desktop app

- Status: accepted
- Date: 2026-09-01

## Context

Commonplace is the first app in this repo that is used at a desk more than on a phone. A
commonplace book earns its keep by making capture cheap, and capture is only cheap if the app
is one keystroke away from whatever you were doing. A browser tab cannot be summoned from
inside another window — you have to find it first, and by then you are managing windows
instead of writing down the thought.

The user asked for a real application on Arch Linux: an entry in the launcher, an icon in the
dock, a global shortcut. `apps/notes` already builds to web (`expo export --platform web`,
`output: "static"`), so the question is what to put around that export, not whether to write
the screens a second time.

## Decision

Tauri v2, in `apps/notes/desktop/`, wrapping `apps/notes/dist`.

`frontendDist` points at the Expo web export; `devUrl` points at Metro on `:8081` so
`tauri dev` is a window onto the normal dev server with fast refresh intact. The Rust side is
one file: it registers **Ctrl+Shift+Space** with the OS via `tauri-plugin-global-shortcut`,
and on press shows, unminimises and focuses the window, then navigates it to the capture
route. That shortcut is the only behaviour in the shell that the web export cannot provide
for itself, and it is the only reason the shell exists.

The webview is granted no capabilities. The bundle imports no `@tauri-apps/api` and calls no
command, so the shortcut is driven entirely from Rust and the shell navigates rather than
emitting an event the web side would have to listen for.

## Alternatives

- **Installable PWA** — zero new toolchain, and Chromium on Arch will give it a launcher entry
  and its own window. It cannot register an OS-level shortcut, which is the one thing being
  asked for. It also makes the browser a runtime dependency of a family application.
- **Electron** — the best-understood option and the one with the fewest surprises, but it ships
  a Chromium per app (~150 MB against Tauri's single-digit MB against the system webview) and
  brings a second Node runtime into a repo that already pins one for Metro. Nothing in this app
  needs a browser engine we control the version of.
- **React Native for Windows/macOS, or Expo on desktop** — no Linux target, which is the only
  target that matters here.
- **No desktop app; keep the browser tab** — the status quo, and the honest baseline. Rejected
  only because of the shortcut.

## Consequences

- **This adds a Rust toolchain to the stack table, which is a human-gated change.** `rustc`,
  `cargo` and `webkit2gtk-4.1` become build prerequisites for one directory. Nothing else in
  the repo compiles Rust, `just verify` does not build it, and CI does not either — the desktop
  bundle is built by hand on the machine that installs it (`apps/notes/desktop/packaging/PKGBUILD`).
  A contributor who never touches `apps/notes/desktop/` never installs any of it.
- **The shell is gated by `just check-desktop`, not by `just verify`.** That target runs
  `cargo fmt --check` and `cargo clippy -D warnings` over `src-tauri`, and it *skips* — exit 0
  with a message — when `cargo` is absent, so a Rust-less checkout is not punished for code it
  cannot compile. Under `CI=true` a missing toolchain is a failure instead, and the `desktop`
  job in `.github/workflows/verify.yml` installs the toolchain and webkit2gtk-4.1 and runs it.
  So the Rust side does have a gate that turns red, and it stays out of the repo-wide one for
  the same reason `lint-go` and `vuln` do: a check that passes because its tool is missing is
  worse than no check. The gate gets no further than `cargo check`/`clippy` — it does not
  bundle, so it needs neither `apps/notes/dist` nor a pnpm install.
- **The escape hatch is that the web export still runs standalone.** `apps/notes/dist` is a
  plain static bundle; serving it from any web server gives the whole app minus the global
  shortcut. If Tauri becomes a burden, deleting `apps/notes/desktop/` costs nothing that is
  not recoverable by opening a browser.
- Keeping that escape hatch open is a rule, not an aspiration: the moment the app imports
  `@tauri-apps/api`, the web build stops being a complete product. Any future native surface —
  tray, notifications, file drops — has to be weighed against that, which is why the capability
  file grants nothing today.
- One more artifact to build and release by hand, on a repo whose other clients ship as an APK
  built by hand as well. Consistent with `just build-apk`, not worse than it.
- **v1 ships no image blocks, on the desktop shell as much as on the phone**, so the desktop
  app needs no file-drop or clipboard-image surface — which is convenient, because either
  would be the first native capability granted to the webview and would cost the escape hatch
  above. The reason images are out is not a desktop one: `libs/go/storage` sets an
  anonymous-read policy on every bucket it creates, so a photo in a never-shared note would be
  fetchable by anyone holding the URL and unsharing would not revoke it. See
  docs/architecture.md and the `UploadNoteImage` comment in `libs/proto/notes/v1/notes.proto`.
- The design canvas is 1240×790 and the window opens at exactly that. The minimum is
  **1024×600**, and the width is not a taste judgement: `DESKTOP_MIN_WIDTH` in
  `apps/notes/app/(app)/_layout.tsx` is 1024, and below it the shell drops to the phone
  tab-bar layout. A 900px minimum let a user drag the window into a 900–1023px band where a
  decorated desktop window rendered the mobile navigation — so the window's floor is the
  layout's breakpoint, and the two are one number by construction rather than by coincidence.
  Changing `DESKTOP_MIN_WIDTH` means changing this too.
