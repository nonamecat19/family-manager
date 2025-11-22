import {
  Controller,
  Get,
  Post,
  Put,
  Delete,
  Body,
  Param,
  Query,
  UseGuards,
} from '@nestjs/common';
import {
  ApiTags,
  ApiOperation,
  ApiResponse,
  ApiParam,
  ApiQuery,
  ApiBearerAuth,
} from '@nestjs/swagger';
import { ListsService } from './lists.service';
import { CreateListDto } from './dto/create-list.dto';
import { UpdateListDto } from './dto/update-list.dto';
import { CreateListItemDto } from './dto/create-list-item.dto';
import { UpdateListItemDto } from './dto/update-list-item.dto';
import { JwtAuthGuard } from '../common/guards/jwt-auth.guard';
import { CurrentUser, UserPayload } from '../common/decorators/user.decorator';

@ApiTags('lists')
@ApiBearerAuth('JWT-auth')
@Controller('lists')
@UseGuards(JwtAuthGuard)
export class ListsController {
  constructor(private readonly listsService: ListsService) {}

  @Get()
  @ApiOperation({ summary: 'Get all lists' })
  @ApiQuery({ name: 'familyId', required: false, description: 'Filter by family ID' })
  @ApiQuery({ name: 'folderId', required: false, description: 'Filter by folder ID' })
  @ApiQuery({ name: 'assignedTo', required: false, description: 'Filter by assigned user ID' })
  @ApiResponse({ status: 200, description: 'List of lists' })
  findAll(
    @Query('familyId') familyId: string,
    @Query('folderId') folderId: string,
    @Query('assignedTo') assignedTo: string,
    @CurrentUser() user: UserPayload,
  ) {
    return this.listsService.findAll(familyId, folderId, assignedTo, user.id);
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get a list by ID' })
  @ApiParam({ name: 'id', description: 'List ID' })
  @ApiResponse({ status: 200, description: 'List details' })
  @ApiResponse({ status: 404, description: 'List not found' })
  findOne(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.listsService.findOne(id, user.id);
  }

  @Post()
  @ApiOperation({ summary: 'Create a new list' })
  @ApiResponse({ status: 201, description: 'List successfully created' })
  @ApiResponse({ status: 400, description: 'Bad request' })
  create(@Body() createListDto: CreateListDto, @CurrentUser() user: UserPayload) {
    return this.listsService.create(createListDto, user.id);
  }

  @Put(':id')
  @ApiOperation({ summary: 'Update a list' })
  @ApiParam({ name: 'id', description: 'List ID' })
  @ApiResponse({ status: 200, description: 'List successfully updated' })
  @ApiResponse({ status: 404, description: 'List not found' })
  update(
    @Param('id') id: string,
    @Body() updateListDto: UpdateListDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.listsService.update(id, updateListDto, user.id);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Delete a list' })
  @ApiParam({ name: 'id', description: 'List ID' })
  @ApiResponse({ status: 200, description: 'List successfully deleted' })
  @ApiResponse({ status: 404, description: 'List not found' })
  remove(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.listsService.remove(id, user.id);
  }

  @Post(':id/items')
  @ApiOperation({ summary: 'Add an item to a list' })
  @ApiParam({ name: 'id', description: 'List ID' })
  @ApiResponse({ status: 201, description: 'Item successfully added' })
  @ApiResponse({ status: 404, description: 'List not found' })
  createItem(
    @Param('id') id: string,
    @Body() createListItemDto: CreateListItemDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.listsService.createItem(id, createListItemDto, user.id);
  }

  @Put(':id/items/:itemId')
  @ApiOperation({ summary: 'Update a list item' })
  @ApiParam({ name: 'id', description: 'List ID' })
  @ApiParam({ name: 'itemId', description: 'Item ID' })
  @ApiResponse({ status: 200, description: 'Item successfully updated' })
  @ApiResponse({ status: 404, description: 'List or item not found' })
  updateItem(
    @Param('id') id: string,
    @Param('itemId') itemId: string,
    @Body() updateListItemDto: UpdateListItemDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.listsService.updateItem(id, itemId, updateListItemDto, user.id);
  }

  @Delete(':id/items/:itemId')
  @ApiOperation({ summary: 'Remove an item from a list' })
  @ApiParam({ name: 'id', description: 'List ID' })
  @ApiParam({ name: 'itemId', description: 'Item ID' })
  @ApiResponse({ status: 200, description: 'Item successfully removed' })
  @ApiResponse({ status: 404, description: 'List or item not found' })
  removeItem(
    @Param('id') id: string,
    @Param('itemId') itemId: string,
    @CurrentUser() user: UserPayload,
  ) {
    return this.listsService.removeItem(id, itemId, user.id);
  }
}

