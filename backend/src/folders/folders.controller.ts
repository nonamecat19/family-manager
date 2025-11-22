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
import { FoldersService } from './folders.service';
import { CreateFolderDto } from './dto/create-folder.dto';
import { UpdateFolderDto } from './dto/update-folder.dto';
import { JwtAuthGuard } from '../common/guards/jwt-auth.guard';
import { CurrentUser, UserPayload } from '../common/decorators/user.decorator';

@ApiTags('folders')
@ApiBearerAuth('JWT-auth')
@Controller('folders')
@UseGuards(JwtAuthGuard)
export class FoldersController {
  constructor(private readonly foldersService: FoldersService) {}

  @Get()
  @ApiOperation({ summary: 'Get all folders' })
  @ApiQuery({ name: 'familyId', required: false, description: 'Filter by family ID' })
  @ApiQuery({ name: 'type', required: false, description: 'Filter by folder type' })
  @ApiResponse({ status: 200, description: 'List of folders' })
  findAll(
    @Query('familyId') familyId: string,
    @Query('type') type: string,
    @CurrentUser() user: UserPayload,
  ) {
    return this.foldersService.findAll(familyId, type, user.id);
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get a folder by ID' })
  @ApiParam({ name: 'id', description: 'Folder ID' })
  @ApiResponse({ status: 200, description: 'Folder details' })
  @ApiResponse({ status: 404, description: 'Folder not found' })
  findOne(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.foldersService.findOne(id, user.id);
  }

  @Post()
  @ApiOperation({ summary: 'Create a new folder' })
  @ApiResponse({ status: 201, description: 'Folder successfully created' })
  @ApiResponse({ status: 400, description: 'Bad request' })
  create(
    @Body() createFolderDto: CreateFolderDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.foldersService.create(createFolderDto, user.id);
  }

  @Put(':id')
  @ApiOperation({ summary: 'Update a folder' })
  @ApiParam({ name: 'id', description: 'Folder ID' })
  @ApiResponse({ status: 200, description: 'Folder successfully updated' })
  @ApiResponse({ status: 404, description: 'Folder not found' })
  update(
    @Param('id') id: string,
    @Body() updateFolderDto: UpdateFolderDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.foldersService.update(id, updateFolderDto, user.id);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Delete a folder' })
  @ApiParam({ name: 'id', description: 'Folder ID' })
  @ApiResponse({ status: 200, description: 'Folder successfully deleted' })
  @ApiResponse({ status: 404, description: 'Folder not found' })
  remove(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.foldersService.remove(id, user.id);
  }
}

