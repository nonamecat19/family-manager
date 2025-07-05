import { Body, Controller, Get, Post, UseGuards } from '@nestjs/common'
import { AuthService } from './auth.service'
import { LoginDto, RegisterDto } from './dto'
import { JwtAuthGuard } from './jwt.guard'

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

  @UseGuards(JwtAuthGuard)
  @Get('/profile')
  getProfile() {
    return {
      user: {},
    }
  }
}
