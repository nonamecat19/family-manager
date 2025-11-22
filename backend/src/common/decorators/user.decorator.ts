import { createParamDecorator, ExecutionContext } from '@nestjs/common';
import { FastifyRequest } from 'fastify';

export interface UserPayload {
  id: string;
  email: string;
  name?: string | null;
  avatar?: string | null;
}

export const CurrentUser = createParamDecorator(
  (data: unknown, ctx: ExecutionContext): UserPayload => {
    const request = ctx.switchToHttp().getRequest<FastifyRequest>();
    return request.user as UserPayload;
  },
);

