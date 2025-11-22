import {
  Injectable,
  CanActivate,
  ExecutionContext,
  UnauthorizedException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { ConfigService } from '@nestjs/config';
import { FastifyRequest } from 'fastify';
import { DatabaseService } from '../../database/database.service';
import { users } from '../../database/schema';
import { eq } from 'drizzle-orm';

@Injectable()
export class JwtAuthGuard implements CanActivate {
  constructor(
    private jwtService: JwtService,
    private configService: ConfigService,
    private databaseService: DatabaseService,
  ) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const request = context.switchToHttp().getRequest<FastifyRequest>();
    const token = this.extractTokenFromHeader(request);

    if (!token) {
      throw new UnauthorizedException('No token provided');
    }

    try {
      const secret = this.configService.get<string>('JWT_SECRET') || 'your-secret-key';
      const payload = await this.jwtService.verifyAsync(token, {
        secret,
      });

      if (!payload || !payload.userId) {
        throw new UnauthorizedException('Invalid token payload');
      }

      const user = await this.databaseService.db.query.users.findFirst({
        where: eq(users.id, payload.userId),
      });

      if (!user) {
        throw new UnauthorizedException('User not found');
      }

      request.user = {
        id: user.id,
        email: user.email,
        name: user.name,
        avatar: user.avatar,
      };

      return true;
    } catch (error) {
      throw new UnauthorizedException('Invalid or expired token');
    }
  }

  private extractTokenFromHeader(request: FastifyRequest): string | undefined {
    const authHeader = request.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return undefined;
    }
    return authHeader.substring(7);
  }
}

