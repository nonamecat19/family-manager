import { Module } from '@nestjs/common';
import { IotController } from './iot.controller';

@Module({
  controllers: [IotController]
})
export class IotModule {}
