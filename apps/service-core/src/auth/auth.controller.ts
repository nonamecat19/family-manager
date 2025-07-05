import { Body, Controller, Get, Post, Req } from '@nestjs/common'
import { MessagePattern } from '@nestjs/microservices'
import { Request } from 'express'
import { RequireAuth } from '../shared/decorators/require-auth.decorator'
import { AuthService } from './auth.service'
import { LoginDto, RegisterDto } from './dto'

@Controller('auth')
export class AuthController {
  constructor(private authService: AuthService) {}

  @Post('/register')
  async register(@Body() registerDto: RegisterDto) {
    const user = await this.authService.register(registerDto)

    return {
      message: 'User registered successfully',
      user,
    }
  }

  @Post('/login')
  async login(@Body() loginDto: LoginDto) {
    return this.authService.login(loginDto.email, loginDto.password)
  }

  @RequireAuth()
  @Get('/profile')
  getProfile(@Req() req: Request) {
    return {
      user: req.user,
    }
  }

  @MessagePattern({ cmd: 'validate_token' })
  async validateToken(data: { token: string }) {
    return this.authService.validateToken(data.token)
  }
}
