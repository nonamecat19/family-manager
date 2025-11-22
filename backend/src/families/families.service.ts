import {
  Injectable,
  NotFoundException,
  ForbiddenException,
  BadRequestException,
} from '@nestjs/common';
import { DatabaseService } from '../database/database.service';
import { families, familyMembers, users } from '../database/schema';
import { eq, and } from 'drizzle-orm';
import { CreateFamilyDto } from './dto/create-family.dto';
import { UpdateFamilyDto } from './dto/update-family.dto';
import { InviteMemberDto } from './dto/invite-member.dto';
import { UpdateMemberRoleDto } from './dto/update-member-role.dto';
import { WebSocketService } from '../websocket/websocket.service';

@Injectable()
export class FamiliesService {
  constructor(
    private databaseService: DatabaseService,
    private websocketService: WebSocketService,
  ) {}

  async findAll(userId: string) {
    const userFamilies = await this.databaseService.db
      .select({
        id: families.id,
        name: families.name,
        icon: families.icon,
        color: families.color,
        role: familyMembers.role,
        createdAt: families.createdAt,
      })
      .from(familyMembers)
      .innerJoin(families, eq(familyMembers.familyId, families.id))
      .where(eq(familyMembers.userId, userId));

    return userFamilies;
  }

  async findOne(id: string, userId: string) {
    // Check if user is a member of this family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, id),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Get family with members
    const family = await this.databaseService.db.query.families.findFirst({
      where: eq(families.id, id),
      with: {
        members: {
          with: {
            user: {
              columns: {
                id: true,
                email: true,
                name: true,
                avatar: true,
              },
            },
          },
        },
      },
    });

    if (!family) {
      throw new NotFoundException('Family not found');
    }

    return family;
  }

  async create(createFamilyDto: CreateFamilyDto, userId: string) {
    const { name, icon, color } = createFamilyDto;

    // Create family
    const [newFamily] = await this.databaseService.db
      .insert(families)
      .values({
        name,
        icon: icon || null,
        color: color || null,
        createdBy: userId,
      })
      .returning();

    // Add creator as owner
    await this.databaseService.db.insert(familyMembers).values({
      userId,
      familyId: newFamily.id,
      role: 'owner',
    });

    // Broadcast family created event
    this.websocketService.broadcastToFamily(newFamily.id, {
      type: 'family_updated',
      familyId: newFamily.id,
      data: { action: 'created', family: newFamily },
    });

    return newFamily;
  }

  async update(id: string, updateFamilyDto: UpdateFamilyDto, userId: string) {
    // Check if user is owner
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, id),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership || membership.role !== 'owner') {
      throw new ForbiddenException('Only owners can update family');
    }

    const updateData: Partial<typeof families.$inferInsert> = {
      updatedAt: new Date(),
    };

    if (updateFamilyDto.name !== undefined) {
      updateData.name = updateFamilyDto.name;
    }
    if (updateFamilyDto.icon !== undefined) {
      updateData.icon = updateFamilyDto.icon;
    }
    if (updateFamilyDto.color !== undefined) {
      updateData.color = updateFamilyDto.color;
    }

    const [updatedFamily] = await this.databaseService.db
      .update(families)
      .set(updateData)
      .where(eq(families.id, id))
      .returning();

    // Broadcast update
    this.websocketService.broadcastToFamily(id, {
      type: 'family_updated',
      familyId: id,
      data: { action: 'updated', family: updatedFamily },
    });

    return updatedFamily;
  }

  async remove(id: string, userId: string) {
    // Check if user is owner
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, id),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership || membership.role !== 'owner') {
      throw new ForbiddenException('Only owners can delete family');
    }

    // Delete family (cascade will handle members, folders, lists, notes, birthdays)
    await this.databaseService.db.delete(families).where(eq(families.id, id));

    // Broadcast deletion
    this.websocketService.broadcastToFamily(id, {
      type: 'family_updated',
      familyId: id,
      data: { action: 'deleted', familyId: id },
    });

    return { success: true };
  }

  async inviteMember(id: string, inviteMemberDto: InviteMemberDto, userId: string) {
    const { email } = inviteMemberDto;

    // Check if user is owner or member
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, id),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Find user to invite
    const invitedUser = await this.databaseService.db.query.users.findFirst({
      where: eq(users.email, email),
    });

    if (!invitedUser) {
      throw new NotFoundException('User not found');
    }

    // Check if already a member
    const existingMember = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, id),
        eq(familyMembers.userId, invitedUser.id),
      ),
    });

    if (existingMember) {
      throw new BadRequestException('User is already a member');
    }

    // Add as member
    await this.databaseService.db.insert(familyMembers).values({
      userId: invitedUser.id,
      familyId: id,
      role: 'member',
    });

    // Broadcast invitation
    this.websocketService.broadcastToFamily(id, {
      type: 'family_updated',
      familyId: id,
      data: { action: 'member_added', userId: invitedUser.id },
    });

    return { success: true, message: 'User invited successfully' };
  }

  async joinFamily(id: string, userId: string) {
    // Check if already a member
    const existingMember = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, id),
        eq(familyMembers.userId, userId),
      ),
    });

    if (existingMember) {
      throw new BadRequestException('Already a member');
    }

    // Add as member
    await this.databaseService.db.insert(familyMembers).values({
      userId,
      familyId: id,
      role: 'member',
    });

    // Broadcast join
    this.websocketService.broadcastToFamily(id, {
      type: 'family_updated',
      familyId: id,
      data: { action: 'member_added', userId },
    });

    return { success: true };
  }

  async updateMemberRole(
    id: string,
    memberUserId: string,
    updateMemberRoleDto: UpdateMemberRoleDto,
    userId: string,
  ) {
    // Check if requester is owner
    const requesterMembership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, id),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!requesterMembership || requesterMembership.role !== 'owner') {
      throw new ForbiddenException('Only owners can update member roles');
    }

    // Update member role
    await this.databaseService.db
      .update(familyMembers)
      .set({ role: updateMemberRoleDto.role })
      .where(
        and(
          eq(familyMembers.familyId, id),
          eq(familyMembers.userId, memberUserId),
        ),
      );

    // Broadcast update
    this.websocketService.broadcastToFamily(id, {
      type: 'family_updated',
      familyId: id,
      data: { action: 'member_updated', userId: memberUserId, role: updateMemberRoleDto.role },
    });

    return { success: true };
  }

  async removeMember(id: string, memberUserId: string, userId: string) {
    // Check if requester is owner or removing themselves
    const requesterMembership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, id),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!requesterMembership) {
      throw new ForbiddenException('Access denied');
    }

    if (memberUserId !== userId && requesterMembership.role !== 'owner') {
      throw new ForbiddenException('Only owners can remove members');
    }

    // Remove member
    await this.databaseService.db
      .delete(familyMembers)
      .where(
        and(
          eq(familyMembers.familyId, id),
          eq(familyMembers.userId, memberUserId),
        ),
      );

    // Broadcast removal
    this.websocketService.broadcastToFamily(id, {
      type: 'family_updated',
      familyId: id,
      data: { action: 'member_removed', userId: memberUserId },
    });

    return { success: true };
  }

  async switchFamily(familyId: string, userId: string) {
    // Verify user is a member
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Return family info for session management (client-side)
    const family = await this.databaseService.db.query.families.findFirst({
      where: eq(families.id, familyId),
    });

    return { family, success: true };
  }
}


