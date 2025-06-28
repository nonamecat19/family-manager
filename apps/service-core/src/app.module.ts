import { Module } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { DBModule } from '@repo/db'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { AuthModule } from './auth/auth.module'
import { NotificationsModule } from './notifications/notifications.module'
import * as schema from './schema'
import { SettingsModule } from './settings/settings.module'
import { UsersModule } from './users/users.module'

@Module({
  imports: [
    DBModule.forRootAsync({
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
    AuthModule,
    SettingsModule,
    NotificationsModule,
    UsersModule,
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
