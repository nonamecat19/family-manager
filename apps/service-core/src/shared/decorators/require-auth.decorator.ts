import { applyDecorators, UseGuards } from '@nestjs/common'
import { RmqAuthGuard } from '../guards/rmq-auth.guard'

export function RequireAuth() {
  return applyDecorators(UseGuards(RmqAuthGuard))
}
