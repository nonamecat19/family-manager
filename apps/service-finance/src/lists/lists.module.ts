import { Module } from '@nestjs/common'
import { ListsController } from './lists.controller'
import { ListsService } from './lists.service'

@Module({
  providers: [ListsService],
  controllers: [ListsController],
})
export class ListsModule {}
