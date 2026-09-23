export type RegisterBeforeSignOut = (hook: () => Promise<void>) => () => void;

export function unregisterOnSignOut(
  registerBeforeSignOut: RegisterBeforeSignOut,
  currentToken: () => string | null,
  unregister: (token: string) => Promise<unknown>,
  forget: () => void,
): () => void {
  return registerBeforeSignOut(async () => {
    const token = currentToken();
    if (!token) return;
    try {
      await unregister(token);
    } finally {
      forget();
    }
  });
}
