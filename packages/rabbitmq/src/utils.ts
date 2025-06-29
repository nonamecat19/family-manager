import type { INestApplication } from '@nestjs/common/interfaces/nest-application.interface'
import { type RmqOptions, Transport } from '@nestjs/microservices'
import { type ServiceName, serviceConfigs } from './config'
import type { RabbitMQOptions, RmqProviderOptions } from './types'

export function getOptions(
  name: ServiceName,
  options?: RabbitMQOptions,
): RmqProviderOptions {
  const config = serviceConfigs[name]
  return {
    name,
    transport: Transport.RMQ,
    options: {
      queue: config.queue,
      ...config.options,
      ...options,
    },
  }
}

export function registerServiceRabbitMQ(
  app: INestApplication,
  { options }: RmqProviderOptions,
): void {
  app.connectMicroservice<RmqOptions>({
    transport: Transport.RMQ,
    options,
  })
}
