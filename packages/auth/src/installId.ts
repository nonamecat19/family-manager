export interface InstallIdStorage {
  getItemAsync(key: string): Promise<string | null>;
  setItemAsync(key: string, value: string): Promise<void>;
}

export type RandomBytes = (count: number) => Uint8Array;

export const INSTALL_ID_KEY = "fm.install_id";

const INSTALL_ID_BYTES = 16;

export function newInstallId(randomBytes: RandomBytes): string {
  let bytes: Uint8Array;
  try {
    bytes = randomBytes(INSTALL_ID_BYTES);
  } catch {
    return "";
  }
  if (!(bytes instanceof Uint8Array) || bytes.length !== INSTALL_ID_BYTES) return "";
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export async function readOrCreateInstallId(
  storage: InstallIdStorage,
  randomBytes: RandomBytes,
): Promise<string> {
  const existing = await storage.getItemAsync(INSTALL_ID_KEY);
  if (existing) return existing;
  const id = newInstallId(randomBytes);
  if (!id) return "";
  await storage.setItemAsync(INSTALL_ID_KEY, id);
  return id;
}
