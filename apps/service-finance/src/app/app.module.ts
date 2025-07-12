import { Module } from '@nestjs/common'
import { ConfigModule } from '@nestjs/config'
import { ClientsModule } from '@nestjs/microservices'
import { AuthProviderRabbitMQ } from '@repo/rabbitmq'
import { ItemsModule } from '../items/items.module'
import { ListsModule } from '../lists/lists.module'
import { DbModule } from './db.module'
import { RmqModule } from './rmq.module'

@Module({
  imports: [
    DbModule,
    ItemsModule,
    ListsModule,
    RmqModule,
    ConfigModule.forRoot({
      isGlobal: true,
      envFilePath: '.env',
    }),
    ClientsModule.register([
      AuthProviderRabbitMQ({
        urls: [
          {
            protocol: 'amqp',
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
  ],
})
export class AppModule {}
