import {
  Body,
  Controller,
  Delete,
  Get,
  Param,
  ParseIntPipe,
  Post,
} from '@nestjs/common'
import { CreateItemDto } from './dto'
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

  @Delete('/:id')
  deleteItem(@Param('id', ParseIntPipe) id: number) {
    return this.itemsService.deleteOne(id)
  }
}
