export const offlineCopy = {
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
