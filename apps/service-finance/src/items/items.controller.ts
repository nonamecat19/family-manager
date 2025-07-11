import { Body, Controller, Delete, Get, Post } from '@nestjs/common'
import { CreateItemDto } from './dto'
import { DeleteItemDto } from './dto/delete-item.dto'
import { ItemsService } from './items.service'

@Controller('items')
export class ItemsController {
  constructor(private readonly itemsService: ItemsService) {}

  @Get('/')
  getItems() {
    return this.itemsService.getAll()
  }

  @Post('/')
  createItem(@Body() createItemDto: CreateItemDto) {
    return this.itemsService.createOne(createItemDto)
  }

  @Delete('/')
  deleteItem(@Body() deleteItemDto: DeleteItemDto) {
    return this.itemsService.deleteOne(deleteItemDto.id)
  }
}
