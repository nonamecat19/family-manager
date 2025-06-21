import { Injectable } from '@nestjs/common';
import { MessagePattern } from '@nestjs/microservices';

@Injectable()
export class AppService {
  getHello(): string {
    return 'Hello World!';
  }

  getUserAuth(userId: any) {
    return {
      userId,
      isAuthenticated: true,
      roles: ['user'],
      permissions: ['read', 'write'],
      authTime: new Date().toISOString(),
      service: 'auth-service'
    };
  }
}
