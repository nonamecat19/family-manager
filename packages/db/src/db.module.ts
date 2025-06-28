import {
  type DynamicModule,
  type InjectionToken,
  Module,
  type OptionalFactoryDependency,
} from '@nestjs/common'
import { ConfigModule, ConfigService } from '@nestjs/config'
import { drizzle, type NodePgDatabase } from 'drizzle-orm/node-postgres'
import { Pool, type PoolConfig } from 'pg'

export const DRIZZLE = Symbol('drizzle-connection')

type BaseSchema = Record<string, unknown>

export interface DrizzleModuleOptions<T extends BaseSchema> {
  schema: T
  connectionOptions: PoolConfig
}

@Module({})
export class DBModule {
  static forRoot<T extends BaseSchema = BaseSchema>({
    schema,
    connectionOptions,
  }: DrizzleModuleOptions<T>): DynamicModule {
    return {
      module: DBModule,
      imports: [ConfigModule],
      providers: [
        {
          provide: DRIZZLE,
          inject: [ConfigService],
          useFactory: () => {
            const pool = new Pool(connectionOptions)

            return drizzle(pool, { schema }) as NodePgDatabase<T>
          },
        },
      ],
      exports: [DRIZZLE],
      global: true,
    }
  }

  static forRootAsync<T extends BaseSchema = BaseSchema>(options: {
    inject?: (InjectionToken | OptionalFactoryDependency)[]
    useFactory: (
      // biome-ignore lint/suspicious/noExplicitAny: any
      ...args: any[]
    ) => DrizzleModuleOptions<T> | Promise<DrizzleModuleOptions<T>>
  }): DynamicModule {
    return {
      module: DBModule,
      imports: [ConfigModule],
      providers: [
        {
          provide: DRIZZLE,
          inject: [ConfigService, ...(options.inject || [])],
          // biome-ignore lint/suspicious/noExplicitAny: any
          useFactory: async (...args: any[]) => {
            const { connectionOptions, schema } = await options.useFactory(
              ...args,
            )
            const pool = new Pool(connectionOptions)

            return drizzle(pool, { schema }) as NodePgDatabase<T>
          },
        },
      ],
      exports: [DRIZZLE],
      global: true,
    }
  }
}

export type DBType<T extends BaseSchema = BaseSchema> = NodePgDatabase<T>
