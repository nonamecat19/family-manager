import { Inject } from '@nestjs/common'
import { ServiceName } from './config'

export const InjectAuthClient = () => Inject(ServiceName.AUTH)
export const InjectNotificationsClient = () => Inject(ServiceName.NOTIFICATIONS)
