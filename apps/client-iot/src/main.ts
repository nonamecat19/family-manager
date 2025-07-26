import { ValidationPipe } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { NestFactory } from '@nestjs/core'
import { MicroserviceOptions, Transport } from '@nestjs/microservices'
import { AppModule } from './app'

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  app.enableCors()
  app.useGlobalPipes(new ValidationPipe())

  const configService = app.get(ConfigService)

  app.connectMicroservice<MicroserviceOptions>({
    transport: Transport.MQTT,
    options: {
      url: `mqtt://${configService.getOrThrow('MQTT_HOST')}:${configService.getOrThrow('MQTT_PORT')}`,
    },
  })

  await app.startAllMicroservices()
  await app.listen(8005)
}
void bootstrap()
