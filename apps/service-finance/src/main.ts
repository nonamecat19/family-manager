import { ValidationPipe } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { NestFactory } from '@nestjs/core'
import { AuthProviderRabbitMQ, registerServiceRabbitMQ } from '@repo/rabbitmq'
import { AppModule } from './app'

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  app.enableCors()
  app.useGlobalPipes(new ValidationPipe())

  const configService = app.get(ConfigService)

  const authRmqProvider = AuthProviderRabbitMQ({
    urls: [
      {
        protocol: configService.get<string>('RABBITMQ_PROTOCOL', 'amqp'),
        username: configService.getOrThrow<string>('RABBITMQ_USERNAME'),
        password: configService.getOrThrow<string>('RABBITMQ_PASSWORD'),
        hostname: configService.getOrThrow<string>(
          'RABBITMQ_HOST',
          'localhost',
        ),
        port: configService.getOrThrow<number>('RABBITMQ_PORT', 5672),
      },
    ],
    queueOptions: {
      durable: true,
    },
  })
  registerServiceRabbitMQ(app, authRmqProvider)

  await app.listen(9002)
}
void bootstrap()
