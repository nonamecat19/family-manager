import { Module } from '@nestjs/common'
import { JwtModule } from '@nestjs/jwt'
import { RmqAuthGuard } from '../shared/guards/rmq-auth.guard'
import { UsersModule } from '../users/users.module'
import { AuthController } from './auth.controller'
import { AuthService } from './auth.service'

@Module({
  controllers: [AuthController],
  providers: [AuthService, RmqAuthGuard],
  imports: [
    JwtModule.register({
      secret: 'your-secret-key',
      signOptions: { expiresIn: '1h' },
    }),
    UsersModule,
  ],
  exports: [RmqAuthGuard],
})
export class AuthModule {}
