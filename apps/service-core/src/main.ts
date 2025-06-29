import { NestFactory } from '@nestjs/core'
import { AuthProviderRabbitMQ, registerServiceRabbitMQ } from '@repo/rabbitmq'
import { AppModule } from './app.module'

async function bootstrap() {
  const app = await NestFactory.create(AppModule)

  const authRmqProvider = AuthProviderRabbitMQ({
    urls: ['amqp://user:password@localhost:5672'],
    queueOptions: {
      durable: true,
    },
  })
  registerServiceRabbitMQ(app, authRmqProvider)

  await app.startAllMicroservices()
  await app.listen(9000)
}
void bootstrap()
