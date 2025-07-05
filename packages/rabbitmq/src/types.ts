import type { RmqOptions } from '@nestjs/microservices'
import type { RmqUrl } from '@nestjs/microservices/external/rmq-url.interface'
import type { ServiceQueue } from './config'

export type RabbitMQOptions = RmqOptions['options'] & {
  urls: string[] | RmqUrl[]
}

export type ServiceConfig = {
  options?: RmqOptions['options']
  queue: ServiceQueue
}

export type ProviderFunc = (options?: RabbitMQOptions) => RmqProviderOptions

export type RmqProviderOptions = RmqOptions & {
  name: string | symbol
}
