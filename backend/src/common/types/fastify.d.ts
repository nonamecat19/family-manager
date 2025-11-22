import { FastifyRequest } from 'fastify';
import { UserPayload } from '../decorators/user.decorator';

declare module 'fastify' {
  interface FastifyRequest {
    user?: UserPayload;
  }
}
