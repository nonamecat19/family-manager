import { Module } from '@nestjs/common'
import { ConfigModule } from '@nestjs/config'
import { PassportModule } from '@nestjs/passport'
import { AuthModule } from '../auth/auth.module'
import { JwtStrategy } from '../auth/jwt.strategy'
import { NotificationsModule } from '../notifications/notifications.module'
import { SettingsModule } from '../settings/settings.module'
import { UsersModule } from '../users/users.module'
import { UsersService } from '../users/users.service'
import { WorkspacesModule } from '../workspaces/workspaces.module'
import { DbModule } from './db.module'
import { RmqModule } from './rmq.module'

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
    }),
    DbModule,
    PassportModule,
    UsersModule,
    AuthModule,
    SettingsModule,
    NotificationsModule,
    WorkspacesModule,
    RmqModule,
  ],
  providers: [JwtStrategy, UsersService],
})
export class AppModule {}
