import {
  Body,
  Controller,
  Delete,
  Get,
  HttpCode,
  HttpStatus,
  Param,
  Post,
  Put,
  Query,
} from '@nestjs/common';
import { ListItemService } from './list-item.service';
import { CreateListItemDto } from './dto/create-list-item.dto';
import { FilterListItemDto } from './dto/filter-list-item.dto';
import { ListItem } from './entities/list-item.entity';

@Controller('list-items')
export class ListItemController {
  constructor(private readonly shoppingListService: ListItemService) {}

  @Get()
  findAll(@Query() filterDto: FilterListItemDto): Promise<ListItem[]> {
    return this.shoppingListService.findAll(filterDto);
  }

  @Get(':id')
  findOne(@Param('id') id: string): Promise<ListItem> {
    return this.shoppingListService.findOne(id);
  }

  @Post()
  create(@Body() createShoppingListDto: CreateListItemDto): Promise<ListItem> {
    return this.shoppingListService.create(createShoppingListDto);
  }

  @Put(':id')
  update(
    @Param('id') id: string,
    @Body() updateShoppingListDto: CreateListItemDto,
  ): Promise<ListItem> {
    return this.shoppingListService.update(id, updateShoppingListDto);
  }

  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  remove(@Param('id') id: string): Promise<void> {
    return this.shoppingListService.remove(id);
  }
}
