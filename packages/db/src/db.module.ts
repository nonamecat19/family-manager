import {DynamicModule, Module} from '@nestjs/common';
import {ConfigModule, ConfigService} from '@nestjs/config';
import {Pool, PoolConfig} from 'pg';
import {drizzle, NodePgDatabase} from 'drizzle-orm/node-postgres';

export const DRIZZLE = Symbol('drizzle-connection');

type BaseSchema = Record<string, unknown>;

export interface DrizzleModuleOptions<T extends BaseSchema> {
    schema: T;
    connectionOptions: PoolConfig
}

@Module({})
export class DBModule {
    static forRoot<T extends BaseSchema = BaseSchema>(
        {schema, connectionOptions}: DrizzleModuleOptions<T>
    ): DynamicModule {
        return {
            module: DBModule,
            imports: [ConfigModule],
            providers: [
                {
                    provide: DRIZZLE,
                    inject: [ConfigService],
                    useFactory: () => {
                        const pool = new Pool(connectionOptions);

                        return drizzle(pool, {schema}) as NodePgDatabase<T>
                    },
                },
            ],
            exports: [DRIZZLE],
            global: true,
        };
    }

    static forRootAsync<T extends BaseSchema = BaseSchema>(options: {
        inject?: any[];
        useFactory: (...args: any[]) => DrizzleModuleOptions<T> | Promise<DrizzleModuleOptions<T>>;
    }): DynamicModule {
        return {
            module: DBModule,
            imports: [ConfigModule],
            providers: [
                {
                    provide: DRIZZLE,
                    inject: [ConfigService, ...(options.inject || [])],
                    useFactory: async (...args: any[]) => {
                        const {connectionOptions, schema} = await options.useFactory(...args);
                        const pool = new Pool(connectionOptions);

                        return drizzle(pool, {schema}) as NodePgDatabase<T>
                    },
                },
            ],
            exports: [DRIZZLE],
            global: true,
        };
    }
}

export type DBType<T extends BaseSchema = BaseSchema> = NodePgDatabase<T>;