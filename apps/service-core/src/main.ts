import { ValidationPipe } from '@nestjs/common'
import { NestFactory } from '@nestjs/core'
import { AuthProviderRabbitMQ, registerServiceRabbitMQ } from '@repo/rabbitmq'
import { AppModule } from './app.module'
import 'dotenv/config'

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  app.useGlobalPipes(new ValidationPipe())

  const authRmqProvider = AuthProviderRabbitMQ({
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
  })
  registerServiceRabbitMQ(app, authRmqProvider)

  await app.startAllMicroservices()
  await app.listen(9000)
}
void bootstrap()
