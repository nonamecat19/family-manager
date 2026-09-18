const ENDPOINTS = {
  production: {
    auth: "https://auth.nonamecat.pp.ua",
    family: "https://family.nonamecat.pp.ua",
    finance: "https://finance.nonamecat.pp.ua",
    recipes: "https://recipes.nonamecat.pp.ua",
    notes: "https://notes.nonamecat.pp.ua",
    notifications: "https://notifications.nonamecat.pp.ua",
    tasks: "https://tasks.nonamecat.pp.ua",
  },
  local: {
    auth: "http://localhost:8081",
    family: "http://localhost:8082",
    finance: "http://localhost:8083",
    recipes: "http://localhost:8084",
    notes: "http://localhost:8085",
    notifications: "http://localhost:8087",
    tasks: "http://localhost:8088",
  },
  emulator: {
    auth: "http://10.0.2.2:8081",
    family: "http://10.0.2.2:8082",
    finance: "http://10.0.2.2:8083",
    recipes: "http://10.0.2.2:8084",
    notes: "http://10.0.2.2:8085",
    notifications: "http://10.0.2.2:8087",
    tasks: "http://10.0.2.2:8088",
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
const recipesUrl = process.env.EXPO_PUBLIC_RECIPES_URL ?? endpoints.recipes;
const notesUrl = process.env.EXPO_PUBLIC_NOTES_URL ?? endpoints.notes;
const tasksUrl = process.env.EXPO_PUBLIC_TASKS_URL ?? endpoints.tasks;
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? tasksUrl;

module.exports = {
  expo: {
    name: "Family Tasks",
    slug: "fm-tasks",
    scheme: "fmtasks",
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
      [
        "expo-auth-session",
        {
          provider: "google",
          iosUrlScheme: "fmtasks",
        },
      ],
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
        recipes: recipesUrl,
        notes: notesUrl,
        tasks: tasksUrl,
      },
    },
    ios: {
      supportsTablet: true,
      bundleIdentifier: "dev.familymanager.tasks",
    },
    android: {
      package: "dev.familymanager.tasks",
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