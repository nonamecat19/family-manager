import { Module } from '@nestjs/common'
import { ConfigModule } from '@nestjs/config'
import { IotModule } from '../iot/iot.module'
import { RmqModule } from './rmq.module'

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
    }),
    IotModule,
    RmqModule,
  ],
})
export class AppModule {}
