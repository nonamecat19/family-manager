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
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? notesUrl;

module.exports = {
  expo: {
    name: "Commonplace",
    slug: "fm-notes",
    scheme: "fmnotes",
    version: "0.1.0",
    orientation: "portrait",
    icon: "./assets/icon.png",
    userInterfaceStyle: "dark",
    backgroundColor: bg,
    newArchEnabled: true,
    plugins: [
      "expo-router",
      "expo-secure-store",
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
        foregroundImage: "./assets/adaptive-icon.png",
        backgroundColor: bg,
      },
      usesCleartextTraffic: true,
    },
    web: {
      bundler: "metro",
      output: "static",
      favicon: "./assets/favicon.png",
    },
  },
};
