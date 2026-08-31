// Dynamic config so the app can be pointed at a different backend without editing code.
// Same shape as apps/recipes/app.config.js — one base URL per service, in both environments.
//
// EXPO_PUBLIC_API_ENV selects the endpoint set and DEFAULTS TO PRODUCTION, so a plain
// `pnpm dev` talks to the deployed services over HTTPS. Point at a local stack explicitly:
//
//   EXPO_PUBLIC_API_ENV=local pnpm dev          # `just up` on this machine
//
// Be aware of what the default means: a development build writes to the REAL database. There
// is one environment, not a staging tier — a transaction deleted while poking at the UI is
// gone for the family too.
//
// Individual URLs can still be overridden, which is what real-device testing needs — localhost
// on a phone resolves to the phone, not to your laptop:
//
//   EXPO_PUBLIC_API_ENV=local EXPO_PUBLIC_FINANCE_URL=http://192.168.1.20:8083 pnpm dev
const ENDPOINTS = {
  production: {
    auth: "https://auth.nonamecat.pp.ua",
    family: "https://family.nonamecat.pp.ua",
    finance: "https://finance.nonamecat.pp.ua",
  },
  local: {
    auth: "http://localhost:8081",
    family: "http://localhost:8082",
    finance: "http://localhost:8083",
  },
};

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
const financeUrl = process.env.EXPO_PUBLIC_FINANCE_URL ?? endpoints.finance;
// finance is this app's own service, so it is also the default for anything unrouted.
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? financeUrl;

module.exports = {
  expo: {
    name: "Family Money",
    slug: "fm-finance",
    scheme: "fmfinance",
    version: "0.1.0",
    orientation: "portrait",
    // Nocturne is a dark-only design system — there is no light pass of these 11 screens, so
    // the app pins the dark scheme rather than following the device and half-rendering.
    userInterfaceStyle: "dark",
    backgroundColor: "#161826",
    newArchEnabled: true,
    plugins: [
      "expo-router",
      "expo-secure-store",
      // Supplies the device locale that seeds the app's language before a choice is stored.
      "expo-localization",
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
        finance: financeUrl,
      },
    },
    ios: {
      supportsTablet: true,
      bundleIdentifier: "dev.familymanager.finance",
    },
    android: {
      package: "dev.familymanager.finance",
      adaptiveIcon: {
        backgroundColor: "#161826",
      },
      // Production is HTTPS and does not need this. It stays for EXPO_PUBLIC_API_ENV=local and
      // LAN testing, which are plain HTTP.
      usesCleartextTraffic: true,
    },
    web: {
      bundler: "metro",
      output: "static",
    },
  },
};
