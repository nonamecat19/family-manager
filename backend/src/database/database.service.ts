import { Injectable, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import * as schema from './schema';

@Injectable()
export class DatabaseService implements OnModuleInit {
  public db: ReturnType<typeof drizzle<typeof schema>>;

  constructor(private configService: ConfigService) {
    const connectionString =
      this.configService.get<string>('DATABASE_URL');

    if (!connectionString) {
      throw new Error('DATABASE_URL or POSTGRES_URL must be set');
    }

    const client = postgres(connectionString);
    this.db = drizzle(client, { schema });
  }

  onModuleInit() {
    // Database connection is established in constructor
  }
}

