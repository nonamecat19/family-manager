import { Module } from '@nestjs/common'
import { ConfigModule } from '@nestjs/config'
import { MqttModule } from '../mqtt/mqtt.module'
import { RmqModule } from './rmq.module'

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
    }),
    RmqModule,
    MqttModule,
  ],
})
export class AppModule {}
