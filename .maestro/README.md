# Maestro mobile testing

[Maestro](https://maestro.mobile.dev) flows for the Family Recipes app (`dev.familymanager.recipes`).

## Prerequisites

1. **Android emulator** running (or a physical device with USB debugging)
2. **Recipes APK installed** — `just build-apk && just install-apk`
3. **Backend running** — `just up` (auth + family + recipes services)

## Run tests

```sh
# All flows
just test-mobile

# A specific flow (without the .yaml extension)
just test-mobile smoke-launch
just test-mobile auth-register
just test-mobile recipe-create
```

## Flow structure

| flow | what it tests |
|---|---|
| `smoke-launch` | app launches, shows the login screen (no backend needed) |
| `auth-register` | register a new account, verify onboarding |
| `auth-login` | sign in with an existing account |
| `recipe-list` | recipes tab shows list or empty state |
| `recipe-create` | create a recipe via the form, verify it appears |
| `recipe-detail` | tap a recipe, verify ingredients/steps/stats |
| `recipe-favorites` | toggle favorite, verify in favorites tab |
| `meal-plan` | navigate to meal plan, quick-add a meal, check ingredients |

## Adding a flow

1. Create `.maestro/flows/recipes/<name>.yaml`
2. Set `appId: dev.familymanager.recipes`
3. Write commands — see the [Maestro command reference](https://docs.maestro.mobile.dev/api-reference)
4. Run `just test-mobile <name>`

## Configuration

The `appId` is set in each flow file (not in a shared config) so flows are self-contained
and can be run individually. The Android package is `dev.familymanager.recipes` (from
`apps/recipes/app.json`).