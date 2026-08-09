# Testing on a real Android device

`localhost` in `EXPO_PUBLIC_API_BASE_URL` (the default) resolves to the *device itself*,
not your dev machine — this is why the app can build and install fine but every API call
fails on a physical phone even though it works on an emulator.

## Option A — USB device, `adb reverse` (recommended)

Keeps the default `localhost` URLs; no rebuild needed if you already have a debug APK.

```
just up                                    # backend
adb reverse tcp:8081 tcp:8081              # auth
adb reverse tcp:8082 tcp:8082              # family
adb reverse tcp:8084 tcp:8084              # recipes
just install-apk
```

## Option B — Wi-Fi device (or `adb reverse` unavailable)

Point the app at your machine's LAN IP instead:

```
export EXPO_PUBLIC_API_BASE_URL=http://<your-lan-ip>:8084
just build-apk       # re-prebuilds (--clean) so the new URL and Android
                      # config (cleartext HTTP) actually take effect
just install-apk
```

Find `<your-lan-ip>` with `ip addr` / `ifconfig`; the phone and dev machine must be on
the same network, and the backend must be reachable there (`docker compose` binds to all
interfaces by default, so this normally just works).

## Verifying

```
just test-mobile smoke-launch     # no backend needed — catches build/render crashes fast
just test-mobile auth-login       # needs backend + a registered test user; the
                                   # `extendedWaitUntil: visible: "Recipes"` step is
                                   # the regression guard for a JS crash post-login
```

If `auth-login` times out waiting for "Recipes", check `just check-ts` first — a missing
export or hook (like the one that caused this app to build but crash on launch) fails
typecheck immediately, before you burn time on a device.
