# Commonplace on the desktop

A [Tauri v2](https://tauri.app) window around the Expo web export of `apps/notes`. It is a
shell, not a port: there is no second codebase here, no React Native for Windows, no separate
build of the screens. `src-tauri/` is ~100 lines of Rust whose entire job is to put the same
web bundle in a real window with a real icon and one keyboard shortcut the browser cannot
give it. The decision and the alternatives are in
[ADR 0009](../../../docs/adr/0009-tauri-desktop.md).

## What is actually different from a browser tab

One thing: **Ctrl+Shift+Space**, registered with the OS, shows and focuses the window and
navigates it to the capture route from wherever you are. That is the whole reason this
directory exists — a commonplace book is only useful if catching a thought costs less
attention than the thought is worth, and hunting for a browser window costs more.

The shortcut lives in `src-tauri/src/lib.rs`. It navigates rather than emitting an event,
which keeps the web bundle free of `@tauri-apps/api` — the same `apps/notes/dist` still runs
in a plain browser with no shell at all.

The route it lands on is `/capture`, and it must exist in `apps/notes/app/`. The shell owns
the key binding; the app owns the screen.

## Prerequisites on Arch

```sh
sudo pacman -S --needed base-devel rust webkit2gtk-4.1 libappindicator-gtk3 librsvg pkgconf
cargo install tauri-cli --version '^2' --locked   # only for `just desktop-dev` / `desktop-build`
```

`webkit2gtk-4.1` is the one to check first when a build fails: Tauri v2 links 4.1, and the
older `webkit2gtk` (4.0) package does not satisfy it. Node and pnpm you already have from the
repo root.

## Running it

```sh
pnpm --filter @fm/app-notes dev    # terminal 1: Metro on :8081
just desktop-dev                   # terminal 2: the window, pointed at Metro
```

Two terminals on purpose. `just desktop-dev` builds no frontend and starts no dev server — if
it did, `tauri dev` would own Metro's lifecycle and you would lose fast refresh every time the
Rust side recompiled. Metro stays yours.

Everything on the app side works exactly as it does on device: fast refresh, the React Query
devtools, `EXPO_PUBLIC_API_ENV=local` to point at `just up` instead of production.

## Building it

```sh
just desktop-build
```

Runs `expo export --platform web` into `apps/notes/dist`, then `tauri build`, and leaves
`.deb`, `.rpm` and AppImage artifacts under `src-tauri/target/release/bundle/`. The web assets
are compiled *into* the binary, so the artifact is self-contained — it does not read
`apps/notes/dist` at runtime.

The first Rust build takes a few minutes and several hundred MB of `target/`. Subsequent ones
are seconds unless you touch `Cargo.toml`.

## Installing it as an Arch package

```sh
cd apps/notes/desktop/packaging && makepkg -si
```

`packaging/PKGBUILD` builds from this checkout — there is no release tarball, and there is one
machine that installs this — and installs `/usr/bin/commonplace`, the `.desktop` entry and the
hicolor icons. It does the Expo export itself, so it does not depend on `just desktop-build`
having run first. The comment block at the top of the PKGBUILD is the authoritative list of
build dependencies.

After the first install, check that the launcher groups the window with its icon:

```sh
xprop WM_CLASS   # then click the Commonplace window
```

If the second string does not match `StartupWMClass` in `packaging/commonplace.desktop`, fix
the `.desktop` file — a mismatch shows up as a duplicate, iconless entry in the dock.

## What is deliberately not here

- **No Tauri commands.** `src-tauri/capabilities/default.json` grants the webview nothing,
  because the webview asks for nothing. Adding a command means adding `@tauri-apps/api` to
  `apps/notes`, which ends the "the same bundle runs in a browser" property. Weigh that first.
- **No auto-updater.** One machine, one `makepkg -si`.
- **No tray icon, no notifications, no deep links.** Each would be a new native surface to keep
  working; none of them is the reason this shell exists.
- **No macOS or Windows bundles.** The icon set carries an `.ico` for the day someone wants
  one, but nothing here has been built or run on either.
