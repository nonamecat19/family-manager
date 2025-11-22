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
import { BirthdaysService } from './birthdays.service';
import { CreateBirthdayDto } from './dto/create-birthday.dto';
import { UpdateBirthdayDto } from './dto/update-birthday.dto';
import { JwtAuthGuard } from '../common/guards/jwt-auth.guard';
import { CurrentUser, UserPayload } from '../common/decorators/user.decorator';

@ApiTags('birthdays')
@ApiBearerAuth('JWT-auth')
@Controller('birthdays')
@UseGuards(JwtAuthGuard)
export class BirthdaysController {
  constructor(private readonly birthdaysService: BirthdaysService) {}

  @Get()
  @ApiOperation({ summary: 'Get all birthdays' })
  @ApiQuery({ name: 'familyId', required: false, description: 'Filter by family ID' })
  @ApiResponse({ status: 200, description: 'List of birthdays' })
  findAll(
    @Query('familyId') familyId: string,
    @CurrentUser() user: UserPayload,
  ) {
    return this.birthdaysService.findAll(familyId, user.id);
  }

  @Get('upcoming')
  @ApiOperation({ summary: 'Get upcoming birthdays' })
  @ApiQuery({ name: 'familyId', required: false, description: 'Filter by family ID' })
  @ApiQuery({ name: 'limit', required: false, description: 'Maximum number of results', type: Number })
  @ApiResponse({ status: 200, description: 'List of upcoming birthdays' })
  findUpcoming(
    @Query('familyId') familyId: string,
    @Query('limit') limit: string,
    @CurrentUser() user: UserPayload,
  ) {
    return this.birthdaysService.findUpcoming(familyId, limit ? parseInt(limit, 10) : 10, user.id);
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get a birthday by ID' })
  @ApiParam({ name: 'id', description: 'Birthday ID' })
  @ApiResponse({ status: 200, description: 'Birthday details' })
  @ApiResponse({ status: 404, description: 'Birthday not found' })
  findOne(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.birthdaysService.findOne(id, user.id);
  }

  @Post()
  @ApiOperation({ summary: 'Create a new birthday' })
  @ApiResponse({ status: 201, description: 'Birthday successfully created' })
  @ApiResponse({ status: 400, description: 'Bad request' })
  create(@Body() createBirthdayDto: CreateBirthdayDto, @CurrentUser() user: UserPayload) {
    return this.birthdaysService.create(createBirthdayDto, user.id);
  }

  @Put(':id')
  @ApiOperation({ summary: 'Update a birthday' })
  @ApiParam({ name: 'id', description: 'Birthday ID' })
  @ApiResponse({ status: 200, description: 'Birthday successfully updated' })
  @ApiResponse({ status: 404, description: 'Birthday not found' })
  update(
    @Param('id') id: string,
    @Body() updateBirthdayDto: UpdateBirthdayDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.birthdaysService.update(id, updateBirthdayDto, user.id);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Delete a birthday' })
  @ApiParam({ name: 'id', description: 'Birthday ID' })
  @ApiResponse({ status: 200, description: 'Birthday successfully deleted' })
  @ApiResponse({ status: 404, description: 'Birthday not found' })
  remove(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.birthdaysService.remove(id, user.id);
  }
}

