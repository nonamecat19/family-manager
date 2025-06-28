import { Controller, Get } from '@nestjs/common'
import { MessagePattern } from '@nestjs/microservices'
import type { AppService } from './app.service'

@Controller()
export class AppController {
  constructor(private readonly appService: AppService) {}

  @Get()
  getHello(): string {
    return this.appService.getHello()
  }

  @MessagePattern({ cmd: 'get_user_auth' })
  // biome-ignore lint/suspicious/noExplicitAny: any
  getUserAuth(any: any) {
    return this.appService.getUserAuth(any)
  }
}
