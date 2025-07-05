import {
  CanActivate,
  ExecutionContext,
  Injectable,
  UnauthorizedException,
} from '@nestjs/common'
import { ClientProxy } from '@nestjs/microservices'
import { InjectAuthClient } from '@repo/rabbitmq'
import { firstValueFrom } from 'rxjs'

@Injectable()
export class RmqAuthGuard implements CanActivate {
  constructor(@InjectAuthClient() private readonly authClient: ClientProxy) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const request = context.switchToHttp().getRequest()
    const authHeader = request.headers.authorization

    if (!authHeader) {
      throw new UnauthorizedException('Authorization header is missing')
    }

    try {
      const [, token] = authHeader.split(' ')
      request.user = await firstValueFrom(
        this.authClient.send({ cmd: 'validate_token' }, { token }),
      )
      return true
    } catch {
      throw new UnauthorizedException('Invalid token')
    }
  }
}
