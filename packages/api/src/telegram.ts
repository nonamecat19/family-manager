export const TELEGRAM_PROVIDER = "telegram";
export const LINK_POLL_MS = 3_000;
export const LINK_WAIT_MS = 10 * 60_000;

const BOT_USERNAME = /^[A-Za-z][A-Za-z0-9_]{4,31}$/;
const START_PAYLOAD = /^[A-Za-z0-9_-]{1,64}$/;

export function normalizeBotUsername(raw: string | null | undefined): string | null {
  const name = (raw ?? "").trim().replace(/^@/, "");
  return BOT_USERNAME.test(name) ? name : null;
}

export function telegramStartUrl(bot: string, token: string): string {
  const name = normalizeBotUsername(bot);
  if (!name) throw new Error(`telegramStartUrl: "${bot}" is not a Telegram bot username`);
  if (!START_PAYLOAD.test(token)) {
    throw new Error("telegramStartUrl: link token does not fit a Telegram start payload");
  }
  return `https://t.me/${name}?start=${token}`;
}
