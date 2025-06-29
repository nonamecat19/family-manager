import { ServiceName } from './config'
import type { ProviderFunc } from './types'
import { getOptions } from './utils'

export const AuthProviderRabbitMQ: ProviderFunc = (options) =>
  getOptions(ServiceName.AUTH, options)

export const NotificationsProviderRabbitMQ: ProviderFunc = (options) =>
  getOptions(ServiceName.NOTIFICATIONS, options)
