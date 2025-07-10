import { Module } from '@nestjs/common'
import { ItemsController } from './items.controller'
import { ItemsService } from './items.service'

@Module({
  providers: [ItemsService],
  controllers: [ItemsController],
})
export class ItemsModule {}
