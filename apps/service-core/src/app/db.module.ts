import { Global, Module } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { DrizzleModule } from '@repo/db'
import * as schema from '../schema'

@Global()
@Module({
  imports: [
    DrizzleModule.forRootAsync({
      inject: [ConfigService],
      useFactory: (configService: ConfigService) => ({
        schema,
        connectionOptions: {
          user: configService.getOrThrow<string>('DB_USER'),
          host: configService.getOrThrow<string>('DB_HOST'),
          database: configService.getOrThrow<string>('DB_NAME'),
          password: configService.getOrThrow<string>('DB_PASSWORD'),
          port: configService.getOrThrow<number>('DB_PORT'),
        },
      }),
    }),
  ],
  exports: [DrizzleModule],
})
export class DbModule {}
