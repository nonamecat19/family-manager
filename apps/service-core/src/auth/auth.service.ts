import { Injectable, UnauthorizedException } from '@nestjs/common'
import { JwtService } from '@nestjs/jwt'
import * as bcrypt from 'bcryptjs'
import { User } from '../schema'
import { UsersService } from '../users/users.service'
import { RegisterDto } from './dto'

export interface JwtPayload {
  sub: number
  email: string
}

@Injectable()
export class AuthService {
  constructor(
    private jwtService: JwtService,
    private usersService: UsersService,
  ) {}

  async register(dto: RegisterDto) {
    await this.usersService.create({
      name: dto.name,
      surname: dto.surname,
      email: dto.email,
      password: await this.getPasswordHash(dto.password),
    })
  }

  async getPasswordHash(password: string): Promise<string> {
    return bcrypt.hash(password, 10)
  }

  async validateUser(email: string, password: string): Promise<User | null> {
    const user = await this.usersService.findByEmail(email)
    if (user && (await bcrypt.compare(password, user.password ?? ''))) {
      return user
    }
    return null
  }

  async login(email: string, password: string) {
    const user = await this.validateUser(email, password)
    if (!user) {
      throw new UnauthorizedException('Invalid credentials')
    }

    const payload: JwtPayload = { sub: user.id, email: user.email }
    return {
      access_token: this.jwtService.sign(payload),
      user: {
        id: user.id,
        email: user.email,
      },
    }
  }

  async validateToken(token: string) {
    try {
      return await this.jwtService.verify(token)
    } catch {
      throw new UnauthorizedException('Invalid token')
    }
  }
}
