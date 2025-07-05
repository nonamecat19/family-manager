import { Module } from '@nestjs/common'
import { ConfigModule } from '@nestjs/config'
import { ClientsModule } from '@nestjs/microservices'
import { AuthProviderRabbitMQ } from '@repo/rabbitmq'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { DrizzleModule } from './db/drizzle.module'
import { TasksModule } from './tasks/tasks.module'

@Module({
  imports: [
    ClientsModule.register([
      AuthProviderRabbitMQ({
        urls: [
          {
            protocol: 'ampq',
            username: 'user',
            password: 'password',
            hostname: 'localhost',
            port: 5672,
          },
        ],
        queueOptions: {
          durable: true,
        },
      }),
    ]),
    ConfigModule,
    DrizzleModule,
    TasksModule,
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
