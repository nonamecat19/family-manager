import { Injectable, OnModuleInit, Logger, OnModuleDestroy } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import * as schema from './schema';

@Injectable()
export class DatabaseService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(DatabaseService.name);
  public db: ReturnType<typeof drizzle<typeof schema>>;
  private client: postgres.Sql;

  constructor(private configService: ConfigService) {
    const connectionString =
      this.configService.get<string>('DATABASE_URL');

    if (!connectionString) {
      throw new Error('DATABASE_URL must be set');
    }

    try {
      this.client = postgres(connectionString, {
        max: 10,
        idle_timeout: 20,
        connect_timeout: 10,
        onnotice: () => {}, // Suppress notices
      });
      
      this.db = drizzle(this.client, { schema });
      this.logger.log('Database client initialized');
    } catch (error) {
      this.logger.error('Failed to initialize database client:', error);
      throw new Error(
        `Failed to connect to database: ${error instanceof Error ? error.message : 'Unknown error'}`,
      );
    }
  }

  async onModuleInit() {
    // Test database connection
    try {
      await this.client`SELECT 1`;
      this.logger.log('Database connection established successfully');
    } catch (error) {
      this.logger.error('Failed to connect to database:', error);
      throw new Error(
        `Database connection failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
      );
    }
  }

  async onModuleDestroy() {
    if (this.client) {
      await this.client.end();
      this.logger.log('Database connection closed');
    }
  }
}

