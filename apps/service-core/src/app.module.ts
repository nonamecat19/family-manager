import { Module } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { PassportModule } from '@nestjs/passport'
import { DBModule } from '@repo/db'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { AuthModule } from './auth/auth.module'
import { JwtStrategy } from './auth/jwt.strategy'
import { NotificationsModule } from './notifications/notifications.module'
import * as schema from './schema'
import { SettingsModule } from './settings/settings.module'
import { UsersModule } from './users/users.module'
import { UsersService } from './users/users.service'
import { WorkspacesModule } from './workspaces/workspaces.module'

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
    PassportModule,
    UsersModule,
    AuthModule,
    SettingsModule,
    NotificationsModule,
    WorkspacesModule,
  ],
  providers: [JwtStrategy, UsersService, AppService],
  controllers: [AppController],
})
export class AppModule {}
