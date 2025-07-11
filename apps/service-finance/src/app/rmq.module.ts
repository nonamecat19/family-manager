import { Global, Module } from '@nestjs/common'
import { ClientsModule } from '@nestjs/microservices'
import { AuthProviderRabbitMQ } from '@repo/rabbitmq'

@Global()
@Module({
  imports: [
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
  exports: [ClientsModule],
})
export class RmqModule {}
