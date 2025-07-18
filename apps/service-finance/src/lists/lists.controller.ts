import {
  Body,
  Controller,
  Delete,
  Get,
  Param,
  ParseIntPipe,
  Post,
} from '@nestjs/common'
import { CreateItemDto } from '../items/dto'
import { DeleteListDto } from './dto'
import { ListsService } from './lists.service'

@Controller('lists')
export class ListsController {
  constructor(private readonly listsService: ListsService) {}

  @Get('/')
  getItems() {
    return this.listsService.getAll()
  }

  @Get('/:id')
  getItemById(@Param('id', ParseIntPipe) id: number) {
    return this.listsService.getById(id)
  }

  @Post('/')
  createItem(@Body() createItemDto: CreateItemDto) {
    return this.listsService.createOne(createItemDto)
  }

  @Delete('/')
  deleteItem(@Body() deleteListDto: DeleteListDto) {
    return this.listsService.deleteOne(
      deleteListDto.id,
      deleteListDto.withItems,
    )
  }
}
