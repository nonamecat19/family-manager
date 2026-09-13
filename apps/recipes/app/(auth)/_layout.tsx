import { Slot } from "expo-router";

import { I18nProvider } from "../../components/i18n/index.tsx";

export default function AuthLayout() {
  return (
    <I18nProvider>
      <Slot />
    </I18nProvider>
  );
}
