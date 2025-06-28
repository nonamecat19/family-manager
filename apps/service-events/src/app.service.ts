import { Inject, Injectable } from '@nestjs/common'
import type { ClientProxy } from '@nestjs/microservices'

@Injectable()
export class AppService {
  constructor(
    @Inject('AUTH_SERVICE') private readonly authClient: ClientProxy,
  ) {}

  getHello(): string {
    return 'Hello World!'
  }

  async getUserAuthData(userId: string) {
    return this.authClient.send({ cmd: 'get_user_auth' }, userId)

    //   try {
    //     const authData = await firstValueFrom(
    //       this.authClient.send({ cmd: 'get_user_auth' }, userId)
    //     );
    //
    //     return {
    //       message: 'User authentication data retrieved successfully',
    //       data: authData,
    //       timestamp: new Date().toISOString(),
    //       service: 'tasks-service'
    //     };
    //   } catch (error) {
    //     console.log(error)
    //     return {
    //       message: 'Failed to retrieve user authentication data',
    //       error: error.message,
    //       timestamp: new Date().toISOString(),
    //       service: 'tasks-service'
    //     };
    //   }
  }
}
