import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import {DBModule} from "@repo/db";
import {ConfigService} from "@nestjs/config";
import { AuthModule } from './auth/auth.module';
import { SettingsModule } from './settings/settings.module';
import { NotificationsModule } from './notifications/notifications.module';
import { UsersModule } from './users/users.module';
import * as schema from './schema'

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
