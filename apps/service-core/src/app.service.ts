import { Injectable } from '@nestjs/common'

@Injectable()
export class AppService {
  getHello(): string {
    return 'Hello World!'
  }

  // biome-ignore lint/suspicious/noExplicitAny: any
  getUserAuth(userId: any) {
    return {
      userId,
      isAuthenticated: true,
      roles: ['user'],
      permissions: ['read', 'write'],
      authTime: new Date().toISOString(),
      service: 'auth-service',
    }
  }
}
