const ENDPOINTS = {
  production: {
    auth: "https://auth.nonamecat.pp.ua",
    family: "https://family.nonamecat.pp.ua",
    finance: "https://finance.nonamecat.pp.ua",
    notifications: "https://notifications.nonamecat.pp.ua",
  },
  local: {
    auth: "http://localhost:8081",
    family: "http://localhost:8082",
    finance: "http://localhost:8083",
    notifications: "http://localhost:8087",
  },
  emulator: {
    auth: "http://10.0.2.2:8081",
    family: "http://10.0.2.2:8082",
    finance: "http://10.0.2.2:8083",
    notifications: "http://10.0.2.2:8087",
  },
};

const organic = require("@fm/config/organic.preset.cjs").theme.extend.colors;

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
const notificationsUrl = process.env.EXPO_PUBLIC_NOTIFICATIONS_URL ?? endpoints.notifications;
const financeUrl = process.env.EXPO_PUBLIC_FINANCE_URL ?? endpoints.finance;
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? financeUrl;

module.exports = {
  expo: {
    name: "Family Money",
    slug: "fm-finance",
    scheme: "fmfinance",
    version: "0.1.0",
    orientation: "portrait",
    icon: "./assets/icon.png",
    userInterfaceStyle: "light",
    backgroundColor: organic.bg,
    newArchEnabled: true,
    plugins: [
      ...(env === "production" ? [] : [require("@fm/config/cleartext.plugin.cjs")]),
      "expo-router",
      "expo-secure-store",
      "expo-localization",
      "expo-notifications",
    ],
    experiments: {
      typedRoutes: true,
    },
    extra: {
      "//": "Endpoints come from EXPO_PUBLIC_API_ENV (default: production). See the top of this file.",
      apiEnv: env,
      telegramBot: process.env.EXPO_PUBLIC_TELEGRAM_BOT ?? "",
      apiBaseUrl,
      serviceUrls: {
        auth: authUrl,
        family: familyUrl,
        notifications: notificationsUrl,
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
        foregroundImage: "./assets/adaptive-icon.png",
        backgroundColor: organic.accent.DEFAULT,
      },
    },
    web: {
      bundler: "metro",
      output: "static",
      favicon: "./assets/favicon.png",
    },
  },
};
