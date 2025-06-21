import {Controller, Get} from '@nestjs/common';
import {AppService} from './app.service';
import {MessagePattern} from "@nestjs/microservices";

@Controller()
export class AppController {
    constructor(private readonly appService: AppService) {
    }

    @Get()
    getHello(): string {
        return this.appService.getHello();
    }

    @MessagePattern({cmd: 'get_user_auth'})
    getUserAuth(any: any) {
        return this.appService.getUserAuth(any)
    }
}
