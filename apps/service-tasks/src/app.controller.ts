import { Controller, Get, Param } from '@nestjs/common';
import { AppService } from './app.service';

@Controller()
export class AppController {
  constructor(private readonly appService: AppService) {}

  @Get()
  getHello(): string {
    return this.appService.getHello();
  }

  @Get('auth/:userId')
  async getUserAuth(@Param('userId') userId: string) {
    return this.appService.getUserAuthData(userId);
  }
}
