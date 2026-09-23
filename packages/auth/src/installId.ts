export interface InstallIdStorage {
  getItemAsync(key: string): Promise<string | null>;
  setItemAsync(key: string, value: string): Promise<void>;
}

export const INSTALL_ID_KEY = "fm.install_id";

const INSTALL_ID_BYTES = 16;

export function newInstallId(): string {
  const bytes = new Uint8Array(INSTALL_ID_BYTES);
  const webcrypto = globalThis.crypto;
  if (webcrypto?.getRandomValues) {
    webcrypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  }
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export async function readOrCreateInstallId(storage: InstallIdStorage): Promise<string> {
  const existing = await storage.getItemAsync(INSTALL_ID_KEY);
  if (existing) return existing;
  const id = newInstallId();
  await storage.setItemAsync(INSTALL_ID_KEY, id);
  return id;
}
