import {
  Controller,
  Get,
  Post,
  Put,
  Delete,
  Body,
  Param,
  UseGuards,
  Query,
} from '@nestjs/common';
import {
  ApiTags,
  ApiOperation,
  ApiResponse,
  ApiParam,
  ApiBearerAuth,
  ApiBody,
} from '@nestjs/swagger';
import { FamiliesService } from './families.service';
import { CreateFamilyDto } from './dto/create-family.dto';
import { UpdateFamilyDto } from './dto/update-family.dto';
import { InviteMemberDto } from './dto/invite-member.dto';
import { UpdateMemberRoleDto } from './dto/update-member-role.dto';
import { SwitchFamilyDto } from '../auth/dto/switch-family.dto';
import { JwtAuthGuard } from '../common/guards/jwt-auth.guard';
import { CurrentUser, UserPayload } from '../common/decorators/user.decorator';

@ApiTags('families')
@ApiBearerAuth('JWT-auth')
@Controller('families')
@UseGuards(JwtAuthGuard)
export class FamiliesController {
  constructor(private readonly familiesService: FamiliesService) {}

  @Get()
  @ApiOperation({ summary: 'Get all families for current user' })
  @ApiResponse({ status: 200, description: 'List of families' })
  findAll(@CurrentUser() user: UserPayload) {
    return this.familiesService.findAll(user.id);
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get a family by ID' })
  @ApiParam({ name: 'id', description: 'Family ID' })
  @ApiResponse({ status: 200, description: 'Family details' })
  @ApiResponse({ status: 404, description: 'Family not found' })
  findOne(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.familiesService.findOne(id, user.id);
  }

  @Post()
  @ApiOperation({ summary: 'Create a new family' })
  @ApiBody({ type: CreateFamilyDto })
  @ApiResponse({ status: 201, description: 'Family successfully created' })
  @ApiResponse({ status: 400, description: 'Bad request' })
  create(
    @Body() createFamilyDto: CreateFamilyDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.familiesService.create(createFamilyDto, user.id);
  }

  @Put(':id')
  @ApiOperation({ summary: 'Update a family' })
  @ApiParam({ name: 'id', description: 'Family ID' })
  @ApiBody({ type: UpdateFamilyDto })
  @ApiResponse({ status: 200, description: 'Family successfully updated' })
  @ApiResponse({ status: 404, description: 'Family not found' })
  update(
    @Param('id') id: string,
    @Body() updateFamilyDto: UpdateFamilyDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.familiesService.update(id, updateFamilyDto, user.id);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Delete a family' })
  @ApiParam({ name: 'id', description: 'Family ID' })
  @ApiResponse({ status: 200, description: 'Family successfully deleted' })
  @ApiResponse({ status: 404, description: 'Family not found' })
  remove(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.familiesService.remove(id, user.id);
  }

  @Post(':id/invite')
  @ApiOperation({ summary: 'Invite a member to a family' })
  @ApiParam({ name: 'id', description: 'Family ID' })
  @ApiBody({ type: InviteMemberDto })
  @ApiResponse({ status: 200, description: 'Member successfully invited' })
  @ApiResponse({ status: 404, description: 'Family not found' })
  inviteMember(
    @Param('id') id: string,
    @Body() inviteMemberDto: InviteMemberDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.familiesService.inviteMember(id, inviteMemberDto, user.id);
  }

  @Post(':id/join')
  @ApiOperation({ summary: 'Join a family by ID' })
  @ApiParam({ name: 'id', description: 'Family ID' })
  @ApiResponse({ status: 200, description: 'Successfully joined family' })
  @ApiResponse({ status: 404, description: 'Family not found' })
  joinFamily(@Param('id') id: string, @CurrentUser() user: UserPayload) {
    return this.familiesService.joinFamily(id, user.id);
  }

  @Put(':id/members/:userId')
  @ApiOperation({ summary: 'Update a member role in a family' })
  @ApiParam({ name: 'id', description: 'Family ID' })
  @ApiParam({ name: 'userId', description: 'User ID' })
  @ApiBody({ type: UpdateMemberRoleDto })
  @ApiResponse({ status: 200, description: 'Member role successfully updated' })
  @ApiResponse({ status: 404, description: 'Family or member not found' })
  updateMemberRole(
    @Param('id') id: string,
    @Param('userId') userId: string,
    @Body() updateMemberRoleDto: UpdateMemberRoleDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.familiesService.updateMemberRole(
      id,
      userId,
      updateMemberRoleDto,
      user.id,
    );
  }

  @Delete(':id/members/:userId')
  @ApiOperation({ summary: 'Remove a member from a family' })
  @ApiParam({ name: 'id', description: 'Family ID' })
  @ApiParam({ name: 'userId', description: 'User ID' })
  @ApiResponse({ status: 200, description: 'Member successfully removed' })
  @ApiResponse({ status: 404, description: 'Family or member not found' })
  removeMember(
    @Param('id') id: string,
    @Param('userId') userId: string,
    @CurrentUser() user: UserPayload,
  ) {
    return this.familiesService.removeMember(id, userId, user.id);
  }

  @Post('switch')
  @ApiOperation({ summary: 'Switch active family' })
  @ApiBody({ type: SwitchFamilyDto })
  @ApiResponse({ status: 200, description: 'Family successfully switched' })
  @ApiResponse({ status: 404, description: 'Family not found' })
  switchFamily(
    @Body() switchFamilyDto: SwitchFamilyDto,
    @CurrentUser() user: UserPayload,
  ) {
    return this.familiesService.switchFamily(switchFamilyDto.familyId, user.id);
  }
}

