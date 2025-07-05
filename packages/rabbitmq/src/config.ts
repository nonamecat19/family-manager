import type { ServiceConfig } from './types'

export enum ServiceName {
  AUTH = 'AUTH_SERVICE',
  NOTIFICATIONS = 'NOTIFICATIONS_SERVICE',
}

export enum ServiceQueue {
  AUTH = 'auth_queue',
  NOTIFICATIONS = 'notifications_queue',
}

export const serviceConfigs: Record<ServiceName, ServiceConfig> = {
  [ServiceName.AUTH]: {
    queue: ServiceQueue.AUTH,
  },
  [ServiceName.NOTIFICATIONS]: {
    queue: ServiceQueue.NOTIFICATIONS,
  },
}
