// Dynamic config so the app can be pointed at a different backend without editing code.
//
// EXPO_PUBLIC_API_ENV selects the endpoint set and DEFAULTS TO PRODUCTION, so a plain
// `pnpm dev` talks to the deployed services over HTTPS. That is the common case: the backend
// is deployed, and most app work does not want a local stack running alongside it. Point at a
// local stack explicitly:
//
//   EXPO_PUBLIC_API_ENV=local pnpm dev          # `just up` on this machine
//
// Be aware of what the default means: a development build writes to the REAL database. There
// is one environment, not a staging tier, so a note deleted while poking at the UI is gone
// for the family too.
//
// Individual URLs can still be overridden, which is what real-device testing needs — localhost
// on a phone resolves to the phone, not to your laptop:
//
//   EXPO_PUBLIC_API_ENV=local EXPO_PUBLIC_NOTES_URL=http://192.168.1.20:8085 pnpm dev
//
// Deployment gives each service its own subdomain (infra/README.md), which is the same shape
// as local development — one base URL per service — so nothing but the hostnames differ.
const ENDPOINTS = {
  production: {
    auth: "https://auth.nonamecat.pp.ua",
    family: "https://family.nonamecat.pp.ua",
    notes: "https://notes.nonamecat.pp.ua",
  },
  local: {
    auth: "http://localhost:8081",
    family: "http://localhost:8082",
    notes: "http://localhost:8085",
  },
};

// The app's ground colour. Native chrome (the splash/window background and the Android
// adaptive icon's ground) is configured here rather than in a stylesheet, but it is the same
// `bg` role as every screen, so it is read from the Tailwind config — the CommonJS mirror of
// components/nocturne/tokens.ts — instead of being re-typed as a hex literal. An app carries
// no colour literals (AGENTS.md, docs/stack.md), and a literal here would drift from the token
// the screens are painted with without anything failing.
const { bg } = require("./tailwind.config.js").theme.extend.colors;

const env = process.env.EXPO_PUBLIC_API_ENV ?? "production";
const endpoints = ENDPOINTS[env];
if (!endpoints) {
  throw new Error(
    `EXPO_PUBLIC_API_ENV must be one of ${Object.keys(ENDPOINTS).join(", ")}, got "${env}". ` +
      `A typo here would otherwise silently fall back to a backend you did not mean to use.`,
  );
}

const authUrl = process.env.EXPO_PUBLIC_AUTH_URL ?? endpoints.auth;
const familyUrl = process.env.EXPO_PUBLIC_FAMILY_URL ?? endpoints.family;
const notesUrl = process.env.EXPO_PUBLIC_NOTES_URL ?? endpoints.notes;
// notes is this app's own service, so it is also the default for anything unrouted.
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? notesUrl;

module.exports = {
  expo: {
    name: "Commonplace",
    slug: "fm-notes",
    scheme: "fmnotes",
    version: "0.1.0",
    orientation: "portrait",
    icon: "./assets/icon.png",
    // Nocturne is a dark-only design system — there is no light pass of these screens, so the
    // app pins the dark scheme rather than following the device and half-rendering.
    userInterfaceStyle: "dark",
    backgroundColor: bg,
    newArchEnabled: true,
    plugins: [
      "expo-router",
      "expo-secure-store",
      // Supplies the device locale that seeds the app's language before a choice is stored.
      "expo-localization",
      // No expo-image-picker: v1 has no image blocks, so the app asks for neither the photo
      // library nor the camera. The reason is server-side — libs/go/storage makes every bucket
      // it creates anonymously readable, which cannot hold a private note's photo — but the
      // permission prompt is the user-visible half of it, and an app that ships a prompt for a
      // feature it does not have is asking for something it cannot justify. See
      // docs/architecture.md.
    ],
    experiments: {
      typedRoutes: true,
    },
    extra: {
      "//": "Endpoints come from EXPO_PUBLIC_API_ENV (default: production). See the top of this file.",
      apiEnv: env,
      apiBaseUrl,
      serviceUrls: {
        auth: authUrl,
        family: familyUrl,
        notes: notesUrl,
      },
    },
    ios: {
      supportsTablet: true,
      bundleIdentifier: "dev.familymanager.notes",
    },
    android: {
      package: "dev.familymanager.notes",
      adaptiveIcon: {
        // Foreground art only: Android draws it over backgroundColor and then masks the
        // result, so the ground behind the mark lives in the colour below, not in the PNG.
        foregroundImage: "./assets/adaptive-icon.png",
        backgroundColor: bg,
      },
      // Production is HTTPS and does not need this. It stays for EXPO_PUBLIC_API_ENV=local and
      // LAN testing, which are plain HTTP: without it, release builds (and any build without
      // the debug manifest override) block cleartext requests.
      usesCleartextTraffic: true,
    },
    web: {
      bundler: "metro",
      output: "static",
      favicon: "./assets/favicon.png",
    },
  },
};
