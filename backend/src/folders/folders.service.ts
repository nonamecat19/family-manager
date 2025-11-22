import {
  Injectable,
  NotFoundException,
  ForbiddenException,
  BadRequestException,
} from '@nestjs/common';
import { DatabaseService } from '../database/database.service';
import { folders, familyMembers } from '../database/schema';
import { eq, and } from 'drizzle-orm';
import { CreateFolderDto } from './dto/create-folder.dto';
import { UpdateFolderDto } from './dto/update-folder.dto';
import { WebSocketService } from '../websocket/websocket.service';

@Injectable()
export class FoldersService {
  constructor(
    private databaseService: DatabaseService,
    private websocketService: WebSocketService,
  ) {}

  async findAll(familyId: string, type: string | undefined, userId: string) {
    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Get folders
    const allFolders = await this.databaseService.db.query.folders.findMany({
      where: and(
        eq(folders.familyId, familyId),
        type ? eq(folders.type, type as 'list' | 'note') : undefined,
      ),
      with: {
        children: true,
      },
    });

    // Filter to only root folders (no parent) and include children
    const rootFolders = allFolders.filter((f) => !f.parentId);

    return rootFolders;
  }

  async findOne(id: string, userId: string) {
    const folder = await this.databaseService.db.query.folders.findFirst({
      where: eq(folders.id, id),
      with: {
        family: true,
        parent: true,
        children: true,
      },
    });

    if (!folder) {
      throw new NotFoundException('Folder not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, folder.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    return folder;
  }

  async create(createFolderDto: CreateFolderDto, userId: string) {
    const { name, icon, color, familyId, type, parentId } = createFolderDto;

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // If parentId is provided, verify it exists and belongs to the same family
    if (parentId) {
      const parent = await this.databaseService.db.query.folders.findFirst({
        where: eq(folders.id, parentId),
      });

      if (!parent || parent.familyId !== familyId) {
        throw new BadRequestException('Invalid parent folder');
      }
    }

    const [newFolder] = await this.databaseService.db
      .insert(folders)
      .values({
        name,
        icon: icon || null,
        color: color || null,
        familyId,
        type,
        parentId: parentId || null,
      })
      .returning();

    // Broadcast folder created
    this.websocketService.broadcastToFamily(familyId, {
      type: 'folder_updated',
      familyId,
      data: { action: 'created', folder: newFolder },
    });

    return newFolder;
  }

  async update(id: string, updateFolderDto: UpdateFolderDto, userId: string) {
    const folder = await this.databaseService.db.query.folders.findFirst({
      where: eq(folders.id, id),
    });

    if (!folder) {
      throw new NotFoundException('Folder not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, folder.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // If parentId is being updated, verify it's valid
    if (updateFolderDto.parentId !== undefined) {
      if (updateFolderDto.parentId === id) {
        throw new BadRequestException('Folder cannot be its own parent');
      }

      if (updateFolderDto.parentId) {
        const parent = await this.databaseService.db.query.folders.findFirst({
          where: eq(folders.id, updateFolderDto.parentId),
        });

        if (!parent || parent.familyId !== folder.familyId) {
          throw new BadRequestException('Invalid parent folder');
        }
      }
    }

    const updateData: Partial<typeof folders.$inferInsert> = {
      updatedAt: new Date(),
    };

    if (updateFolderDto.name !== undefined) {
      updateData.name = updateFolderDto.name;
    }
    if (updateFolderDto.icon !== undefined) {
      updateData.icon = updateFolderDto.icon;
    }
    if (updateFolderDto.color !== undefined) {
      updateData.color = updateFolderDto.color;
    }
    if (updateFolderDto.parentId !== undefined) {
      updateData.parentId = updateFolderDto.parentId;
    }

    const [updatedFolder] = await this.databaseService.db
      .update(folders)
      .set(updateData)
      .where(eq(folders.id, id))
      .returning();

    // Broadcast update
    this.websocketService.broadcastToFamily(folder.familyId, {
      type: 'folder_updated',
      familyId: folder.familyId,
      data: { action: 'updated', folder: updatedFolder },
    });

    return updatedFolder;
  }

  async remove(id: string, userId: string) {
    const folder = await this.databaseService.db.query.folders.findFirst({
      where: eq(folders.id, id),
    });

    if (!folder) {
      throw new NotFoundException('Folder not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, folder.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Check if folder has children
    const children = await this.databaseService.db.query.folders.findMany({
      where: eq(folders.parentId, id),
    });

    if (children.length > 0) {
      throw new BadRequestException('Cannot delete folder with subfolders');
    }

    // Delete folder (cascade will handle lists/notes)
    await this.databaseService.db.delete(folders).where(eq(folders.id, id));

    // Broadcast deletion
    this.websocketService.broadcastToFamily(folder.familyId, {
      type: 'folder_updated',
      familyId: folder.familyId,
      data: { action: 'deleted', folderId: id },
    });

    return { success: true };
  }
}


