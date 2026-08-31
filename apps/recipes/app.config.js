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
// is one environment, not a staging tier, so a recipe deleted while poking at the UI is gone
// for the family too.
//
// Individual URLs can still be overridden, which is what real-device testing needs — localhost
// on a phone resolves to the phone, not to your laptop:
//
//   EXPO_PUBLIC_API_ENV=local EXPO_PUBLIC_RECIPES_URL=http://192.168.1.20:8084 pnpm dev
//
// Deployment gives each service its own subdomain (infra/README.md), which is the same shape
// as local development — one base URL per service — so nothing but the hostnames differ.
const ENDPOINTS = {
  production: {
    auth: "https://auth.nonamecat.pp.ua",
    family: "https://family.nonamecat.pp.ua",
    recipes: "https://recipes.nonamecat.pp.ua",
  },
  local: {
    auth: "http://localhost:8081",
    family: "http://localhost:8082",
    recipes: "http://localhost:8084",
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
const recipesUrl = process.env.EXPO_PUBLIC_RECIPES_URL ?? endpoints.recipes;
// recipes is this app's own service, so it is also the default for anything unrouted.
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? recipesUrl;

module.exports = {
  expo: {
    name: "Family Recipes",
    slug: "fm-recipes",
    scheme: "fmrecipes",
    version: "0.1.0",
    orientation: "portrait",
    icon: "./assets/icon.png",
    userInterfaceStyle: "automatic",
    newArchEnabled: true,
    plugins: [
      "expo-router",
      "expo-secure-store",
      // Supplies the device locale that seeds the app's language before a choice is stored.
      "expo-localization",
      [
        "expo-image-picker",
        {
          photosPermission: "Family Recipes uses your photo library to attach a photo to a recipe.",
        },
      ],
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
        recipes: recipesUrl,
      },
    },
    ios: {
      supportsTablet: true,
      bundleIdentifier: "dev.familymanager.recipes",
    },
    android: {
      package: "dev.familymanager.recipes",
      adaptiveIcon: {
        // Foreground art only: Android draws it over backgroundColor and then masks the
        // result, so the plate and its ring live in the colour below, not in the PNG.
        foregroundImage: "./assets/adaptive-icon.png",
        backgroundColor: "#C4643C",
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
