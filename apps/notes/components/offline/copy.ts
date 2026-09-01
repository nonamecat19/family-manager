/**
 * Copy for the offline queue's new states.
 *
 * It lives here rather than in components/i18n/strings.ts because that file is owned by another
 * change in flight. FOLLOW-UP for whoever owns i18n: fold `offlineCopy` into `strings` (as
 * `strings.capture.*` and `strings.settings.*`) and delete this module — the app should have one
 * copy door, and this is a second one.
 */
export const offlineCopy = {
  /** Shown in the sheet when a capture could not be written at all. */
  captureNotSaved: "Not saved yet — try again in a moment.",
  refusedTitle: "Could not be filed",
  refusedBody: (n: number) =>
    n === 1
      ? "1 note could not be saved where it was captured. Your writing is safe on this device."
      : `${n} notes could not be saved where they were captured. Your writing is safe on this device.`,
  refile: "Save to my notes",
  unsyncedOnSignOut: (n: number) =>
    n === 1
      ? "1 note has not synced. It stays on this device and comes back when you sign in again."
      : `${n} notes have not synced. They stay on this device and come back when you sign in again.`,
  signingOut: "Signing out…",
} as const;
