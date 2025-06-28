import { Module } from '@nestjs/common'
import { ConfigModule } from '@nestjs/config'
import { ClientsModule, Transport } from '@nestjs/microservices'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { DrizzleModule } from './db/drizzle.module'
import { TasksModule } from './tasks/tasks.module'

@Module({
  imports: [
    ClientsModule.register([
      {
        name: 'AUTH_SERVICE',
        transport: Transport.RMQ,
        options: {
          urls: ['amqp://user:password@localhost:5672'],
          queue: 'auth_queue',
          queueOptions: {
            durable: true,
          },
        },
      },
    ]),
    ConfigModule,
    DrizzleModule,
    TasksModule,
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
