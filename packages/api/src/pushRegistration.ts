export interface PushRegistrar {
  resolveToken(): Promise<string | null>;
  register(token: string): Promise<unknown>;
  unregister(token: string): Promise<unknown>;
}

export class PushRegistration {
  private readonly registrar: PushRegistrar;
  private token: string | null = null;
  private key: string | null = null;
  private queue: Promise<void> = Promise.resolve();
  private closed = true;

  constructor(registrar: PushRegistrar) {
    this.registrar = registrar;
  }

  sync(userId: string): Promise<void> {
    this.closed = false;
    this.queue = this.queue.then(() => this.registerFor(userId));
    return this.queue;
  }

  async signOut(): Promise<void> {
    this.closed = true;
    await this.queue;
    const token = this.token;
    this.token = null;
    this.key = null;
    if (token) await this.registrar.unregister(token);
  }

  reset(): void {
    this.closed = true;
    this.token = null;
    this.key = null;
  }

  private async registerFor(userId: string): Promise<void> {
    if (this.closed) return;
    const token = await this.registrar.resolveToken().catch(() => null);
    if (!token || this.closed) return;

    const key = `${userId}:${token}`;
    if (this.key === key) return;

    this.key = key;
    this.token = token;
    try {
      await this.registrar.register(token);
    } catch {
      if (this.key === key) this.key = null;
    }
  }
}
