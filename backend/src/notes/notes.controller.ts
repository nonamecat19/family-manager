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
  Req,
} from '@nestjs/common';
import {
  ApiTags,
  ApiOperation,
  ApiResponse,
  ApiParam,
  ApiQuery,
  ApiBearerAuth,
  ApiConsumes,
} from '@nestjs/swagger';
import { FastifyRequest } from 'fastify';
import { NotesService } from './notes.service';
import { CreateNoteDto } from './dto/create-note.dto';
import { UpdateNoteDto } from './dto/update-note.dto';
import { JwtAuthGuard } from '../common/guards/jwt-auth.guard';
import { CurrentUser, UserPayload } from '../common/decorators/user.decorator';

@ApiTags('notes')
@ApiBearerAuth('JWT-auth')
@Controller('notes')
@UseGuards(JwtAuthGuard)
export class NotesController {
  constructor(private readonly notesService: NotesService) {}

  @Get()
  @ApiOperation({ summary: 'Get all notes' })
  @ApiQuery({ name: 'familyId', required: false, description: 'Filter by family ID' })
  @ApiQuery({ name: 'folderId', required: false, description: 'Filter by folder ID' })
  @ApiResponse({ status: 200, description: 'List of notes' })
  findAll(
    @Query('familyId') familyId: string,
    @Query('folderId') folderId: string,
    @CurrentUser() user: UserPayload,
  ) {
    return this.notesService.findAll(familyId, folderId, user.id);
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get a note by ID' })
  @ApiParam({ name: 'id', description: 'Note ID' })
  @ApiResponse({ status: 200, description: 'Note details' })
  @ApiResponse({ status: 404, description: 'Note not found' })
  findOne(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.notesService.findOne(id, user.id);
  }

  @Post()
  @ApiOperation({ summary: 'Create a new note' })
  @ApiResponse({ status: 201, description: 'Note successfully created' })
  @ApiResponse({ status: 400, description: 'Bad request' })
  create(@Body() createNoteDto: CreateNoteDto, @CurrentUser() user: UserPayload) {
    return this.notesService.create(createNoteDto, user.id);
  }

  @Put(':id')
  @ApiOperation({ summary: 'Update a note' })
  @ApiParam({ name: 'id', description: 'Note ID' })
  @ApiResponse({ status: 200, description: 'Note successfully updated' })
  @ApiResponse({ status: 404, description: 'Note not found' })
  update(
    @Param('id') id: string,
    @Body() updateNoteDto: UpdateNoteDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.notesService.update(id, updateNoteDto, user.id);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Delete a note' })
  @ApiParam({ name: 'id', description: 'Note ID' })
  @ApiResponse({ status: 200, description: 'Note successfully deleted' })
  @ApiResponse({ status: 404, description: 'Note not found' })
  remove(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.notesService.remove(id, user.id);
  }

  @Post(':id/upload')
  @ApiOperation({ summary: 'Upload a file to a note' })
  @ApiParam({ name: 'id', description: 'Note ID' })
  @ApiConsumes('multipart/form-data')
  @ApiResponse({ status: 200, description: 'File successfully uploaded' })
  @ApiResponse({ status: 404, description: 'Note not found' })
  async uploadFile(
    @Param('id') id: string,
    @Req() request: FastifyRequest,
    @CurrentUser() user: UserPayload,
  ) {
    const data = await request.file();
    if (!data) {
      throw new Error('No file provided');
    }

    const buffer = await data.toBuffer();
    const filename = data.filename || (request.body as any)?.filename || 'file';

    return this.notesService.uploadFile(id, buffer, filename, user.id);
  }
}

