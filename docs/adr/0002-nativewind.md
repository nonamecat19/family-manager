# ADR 0002 — NativeWind for packages/ui

- Status: accepted
- Date: 2026-08-06

## Context

Three Expo apps share one design system. The styling layer decides how `packages/ui`,
`packages/theme` and per-app build config look, and it is expensive to change later.

## Decision

NativeWind (Tailwind class names compiled to React Native styles). Tokens — colors, spacing,
radii, typography — live in a Tailwind preset exported by `packages/theme`; each app's
`tailwind.config.js` extends that preset through `packages/config`. `packages/ui` exports
components that accept `className`.

## Alternatives

- **Tamagui** — better list performance and a real theming runtime, but requires a babel plugin
  and metro config in every app, and its compiler is another moving part in the Turborepo cache.
  Revisit if profiling shows style cost in long lists.
- **Plain StyleSheet + token objects** — zero dependencies, but every component re-implements
  variants by hand.

## Consequences

- One token source; a hard-coded color in an app is a review reject.
- Metro/babel config for NativeWind lives in `packages/config` only. Forking it per app is the
  usual cause of "styles work in one app".
- Web support (if an app ever ships to web) comes free from Tailwind semantics.
