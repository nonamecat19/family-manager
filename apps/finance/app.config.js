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
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? financeUrl;

module.exports = {
  expo: {
    name: "Family Money",
    slug: "fm-finance",
    scheme: "fmfinance",
    version: "0.1.0",
    orientation: "portrait",
    icon: "./assets/icon.png",
    userInterfaceStyle: "dark",
    backgroundColor: "#161826",
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
        backgroundColor: "#C4643C",
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
