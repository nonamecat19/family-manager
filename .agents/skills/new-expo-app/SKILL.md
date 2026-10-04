---
name: new-expo-app
description: Scaffold a new Expo app under apps/ wired to the shared packages (ui, auth, api, theme, config) and the generated TypeScript SDK, registered with pnpm workspaces and Turborepo, then verified against the repo knowledge graph. Use when adding a mobile app or a shared TS package in this monorepo.
---

# new-expo-app

## Scaffold

```sh
pnpm create expo-app apps/<name> --template blank-typescript
cd apps/<name>
```

Then make it a workspace citizen:

- `package.json` name: `@fm/<name>`, `"private": true`
- add scripts Turborepo expects: `dev` (`expo start`), `build`, `lint`, `typecheck`
  (`tsc --noEmit`), `test`
- depend on shared packages with `workspace:*`:
  ```json
  {
    "dependencies": {
      "@fm/ui": "workspace:*",
      "@fm/auth": "workspace:*",
      "@fm/api": "workspace:*",
      "@fm/theme": "workspace:*"
    },
    "devDependencies": { "@fm/config": "workspace:*" }
  }
  ```
- extend presets from `@fm/config`: `tsconfig.json`, eslint, babel (NativeWind plugin), metro,
  and `tailwind.config.js` extending the `@fm/theme` preset. Do not fork configs into the app —
  a diverging metro/babel config is the usual cause of "works in one app only".
- navigation is **expo-router** (file-based, `app/` directory); no react-navigation config by
  hand.
- metro must resolve the workspace root: enable `watchFolders` for the repo root and
  `disableHierarchicalLookup: false`, or shared packages will not hot-reload.

`apps/*` is already covered by `pnpm-workspace.yaml` and `turbo.json` — no registration edit
needed. Run `pnpm install` from the repo root, never from inside the app.

## Boundaries

- Apps hold **screens and navigation only.** Anything reusable moves to `packages/ui`.
- No app imports another app.
- All service calls go through `packages/api`, which wraps generated `sdk/typescript` with
  `createClient` (ConnectRPC) + TanStack Query hooks. An app never imports `sdk/typescript`
  directly and never hand-writes a fetch to a service.
- Styling is NativeWind classes against the `@fm/theme` Tailwind preset. A literal color, font
  size or spacing value in an app is a review reject.
- Tokens/sessions live in expo-secure-store via `@fm/auth` — never AsyncStorage, never a store.

## New shared package instead

Same rules, in `packages/<name>`: `@fm/<name>`, `"main": "src/index.ts"` (source-first, Metro
transpiles), a single public entry point, no deep imports from consumers.

## Verify

```sh
pnpm install
just graph          # app:<name> node + DEPENDS_ON edges to each @fm/* package must appear
just check-ts       # turbo lint + typecheck + test
just impact pkg:@fm/ui    # sanity: the new app shows up as a dependent
```

If the app node appears with no `DEPENDS_ON` edges, the shared deps are not `workspace:*` —
fix that before writing screens.

## Parallelism

Screens/navigation and API integration run in parallel **only if the contract is unchanged**.
If this app needs a new rpc, that is a contract change: sequential, one agent, see the
`contract-change` skill.
