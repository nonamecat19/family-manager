import { Module } from '@nestjs/common'
import { ConfigModule, ConfigService } from '@nestjs/config'
import { drizzle, type NodePgDatabase } from 'drizzle-orm/node-postgres'
import { Pool } from 'pg'
import * as schema from './schema'

const DRIZZLE = Symbol('db-connection')

@Module({
  imports: [ConfigModule],
  providers: [
    {
      provide: DRIZZLE,
      inject: [ConfigService],
      useFactory: (configService: ConfigService) => {
        const databaseURL = configService.get<string>('DATABASE_URL')
        const pool = new Pool({
          connectionString: databaseURL,
          ssl: false,
        })
        return drizzle(pool, { schema }) as NodePgDatabase<typeof schema>
      },
    },
  ],
  exports: [DRIZZLE],
})
export class DrizzleModule {}
