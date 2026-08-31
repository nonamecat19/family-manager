import { Slot } from "expo-router";

import { I18nProvider } from "../../components/i18n/index.tsx";

/** Mirrors the (app) group's own provider mount: the auth group has no other layout, so this
 * is the only place `login.tsx` can pick up a locale before the family gate exists. */
export default function AuthLayout() {
  return (
    <I18nProvider>
      <Slot />
    </I18nProvider>
  );
}
