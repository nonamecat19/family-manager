import {
  Injectable,
  NotFoundException,
  ForbiddenException,
  BadRequestException,
} from '@nestjs/common';
import { DatabaseService } from '../database/database.service';
import { lists, listItems, familyMembers, folders } from '../database/schema';
import { eq, and, asc, desc } from 'drizzle-orm';
import { CreateListDto } from './dto/create-list.dto';
import { UpdateListDto } from './dto/update-list.dto';
import { CreateListItemDto } from './dto/create-list-item.dto';
import { UpdateListItemDto } from './dto/update-list-item.dto';
import { WebSocketService } from '../websocket/websocket.service';

@Injectable()
export class ListsService {
  constructor(
    private databaseService: DatabaseService,
    private websocketService: WebSocketService,
  ) {}

  async findAll(familyId: string, folderId: string | undefined, assignedTo: string | undefined, userId: string) {
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

    // Build query conditions
    const conditions = [eq(lists.familyId, familyId)];

    if (folderId) {
      conditions.push(eq(lists.folderId, folderId));
    }

    if (assignedTo) {
      conditions.push(eq(lists.assignedTo, assignedTo));
    }

    const userLists = await this.databaseService.db.query.lists.findMany({
      where: and(...conditions),
      with: {
        folder: true,
        assignedUser: {
          columns: {
            id: true,
            name: true,
            email: true,
            avatar: true,
          },
        },
        items: {
          orderBy: (items, { asc }) => [asc(items.order)],
        },
      },
      orderBy: (lists, { desc }) => [desc(lists.createdAt)],
    });

    return userLists;
  }

  async findOne(id: string, userId: string) {
    const list = await this.databaseService.db.query.lists.findFirst({
      where: eq(lists.id, id),
      with: {
        family: true,
        folder: true,
        assignedUser: {
          columns: {
            id: true,
            name: true,
            email: true,
            avatar: true,
          },
        },
        items: {
          orderBy: (items, { asc }) => [asc(items.order)],
        },
      },
    });

    if (!list) {
      throw new NotFoundException('List not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, list.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    return list;
  }

  async create(createListDto: CreateListDto, userId: string) {
    const { title, description, folderId, assignedTo, dueDate, dueTime, familyId } = createListDto;

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

    // If folderId is provided, verify it exists and belongs to the family
    if (folderId) {
      const folder = await this.databaseService.db.query.folders.findFirst({
        where: eq(folders.id, folderId),
      });

      if (!folder || folder.familyId !== familyId || folder.type !== 'list') {
        throw new BadRequestException('Invalid folder');
      }
    }

    // If assignedTo is provided, verify user is a member of the family
    if (assignedTo) {
      const assignedMembership = await this.databaseService.db.query.familyMembers.findFirst({
        where: and(
          eq(familyMembers.familyId, familyId),
          eq(familyMembers.userId, assignedTo),
        ),
      });

      if (!assignedMembership) {
        throw new BadRequestException('Assigned user is not a member of the family');
      }
    }

    const [newList] = await this.databaseService.db
      .insert(lists)
      .values({
        title,
        description: description || null,
        folderId: folderId || null,
        assignedTo: assignedTo || null,
        dueDate: dueDate || null,
        dueTime: dueTime || null,
        familyId,
        completed: false,
      })
      .returning();

    // Broadcast list created
    this.websocketService.broadcastToFamily(familyId, {
      type: 'list_updated',
      familyId,
      data: { action: 'created', list: newList },
    });

    return newList;
  }

  async update(id: string, updateListDto: UpdateListDto, userId: string) {
    const list = await this.databaseService.db.query.lists.findFirst({
      where: eq(lists.id, id),
    });

    if (!list) {
      throw new NotFoundException('List not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, list.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Validate folder if being updated
    if (updateListDto.folderId !== undefined && updateListDto.folderId) {
      const folder = await this.databaseService.db.query.folders.findFirst({
        where: eq(folders.id, updateListDto.folderId),
      });

      if (!folder || folder.familyId !== list.familyId || folder.type !== 'list') {
        throw new BadRequestException('Invalid folder');
      }
    }

    // Validate assignedTo if being updated
    if (updateListDto.assignedTo !== undefined && updateListDto.assignedTo) {
      const assignedMembership = await this.databaseService.db.query.familyMembers.findFirst({
        where: and(
          eq(familyMembers.familyId, list.familyId),
          eq(familyMembers.userId, updateListDto.assignedTo),
        ),
      });

      if (!assignedMembership) {
        throw new BadRequestException('Assigned user is not a member of the family');
      }
    }

    const updateData: Partial<typeof lists.$inferInsert> = {
      updatedAt: new Date(),
    };

    if (updateListDto.title !== undefined) {
      updateData.title = updateListDto.title;
    }
    if (updateListDto.description !== undefined) {
      updateData.description = updateListDto.description;
    }
    if (updateListDto.folderId !== undefined) {
      updateData.folderId = updateListDto.folderId;
    }
    if (updateListDto.assignedTo !== undefined) {
      updateData.assignedTo = updateListDto.assignedTo;
    }
    if (updateListDto.dueDate !== undefined) {
      updateData.dueDate = updateListDto.dueDate;
    }
    if (updateListDto.dueTime !== undefined) {
      updateData.dueTime = updateListDto.dueTime;
    }
    if (updateListDto.completed !== undefined) {
      updateData.completed = updateListDto.completed;
    }

    const [updatedList] = await this.databaseService.db
      .update(lists)
      .set(updateData)
      .where(eq(lists.id, id))
      .returning();

    // Broadcast update
    this.websocketService.broadcastToFamily(list.familyId, {
      type: 'list_updated',
      familyId: list.familyId,
      data: { action: 'updated', list: updatedList },
    });

    return updatedList;
  }

  async remove(id: string, userId: string) {
    const list = await this.databaseService.db.query.lists.findFirst({
      where: eq(lists.id, id),
    });

    if (!list) {
      throw new NotFoundException('List not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, list.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Delete list (cascade will handle items)
    await this.databaseService.db.delete(lists).where(eq(lists.id, id));

    // Broadcast deletion
    this.websocketService.broadcastToFamily(list.familyId, {
      type: 'list_updated',
      familyId: list.familyId,
      data: { action: 'deleted', listId: id },
    });

    return { success: true };
  }

  async createItem(listId: string, createListItemDto: CreateListItemDto, userId: string) {
    const { content, order } = createListItemDto;

    const list = await this.databaseService.db.query.lists.findFirst({
      where: eq(lists.id, listId),
    });

    if (!list) {
      throw new NotFoundException('List not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, list.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Get max order if not provided
    let itemOrder = order;
    if (itemOrder === undefined) {
      const maxOrderItem = await this.databaseService.db.query.listItems.findFirst({
        where: eq(listItems.listId, listId),
        orderBy: (items, { desc }) => [desc(items.order)],
      });
      itemOrder = maxOrderItem ? maxOrderItem.order + 1 : 0;
    }

    const [newItem] = await this.databaseService.db
      .insert(listItems)
      .values({
        listId,
        content,
        order: itemOrder,
        completed: false,
      })
      .returning();

    // Broadcast item added
    this.websocketService.broadcastToFamily(list.familyId, {
      type: 'list_updated',
      familyId: list.familyId,
      data: { action: 'item_added', listId, item: newItem },
    });

    return newItem;
  }

  async updateItem(listId: string, itemId: string, updateListItemDto: UpdateListItemDto, userId: string) {
    const list = await this.databaseService.db.query.lists.findFirst({
      where: eq(lists.id, listId),
    });

    if (!list) {
      throw new NotFoundException('List not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, list.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Verify item belongs to list
    const item = await this.databaseService.db.query.listItems.findFirst({
      where: and(eq(listItems.id, itemId), eq(listItems.listId, listId)),
    });

    if (!item) {
      throw new NotFoundException('Item not found');
    }

    const updateData: Partial<typeof listItems.$inferInsert> = {
      updatedAt: new Date(),
    };

    if (updateListItemDto.content !== undefined) {
      updateData.content = updateListItemDto.content;
    }
    if (updateListItemDto.order !== undefined) {
      updateData.order = updateListItemDto.order;
    }
    if (updateListItemDto.completed !== undefined) {
      updateData.completed = updateListItemDto.completed;
    }

    const [updatedItem] = await this.databaseService.db
      .update(listItems)
      .set(updateData)
      .where(eq(listItems.id, itemId))
      .returning();

    // Broadcast item updated
    this.websocketService.broadcastToFamily(list.familyId, {
      type: 'list_updated',
      familyId: list.familyId,
      data: { action: 'item_updated', listId, item: updatedItem },
    });

    return updatedItem;
  }

  async removeItem(listId: string, itemId: string, userId: string) {
    const list = await this.databaseService.db.query.lists.findFirst({
      where: eq(lists.id, listId),
    });

    if (!list) {
      throw new NotFoundException('List not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, list.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Verify item belongs to list
    const item = await this.databaseService.db.query.listItems.findFirst({
      where: and(eq(listItems.id, itemId), eq(listItems.listId, listId)),
    });

    if (!item) {
      throw new NotFoundException('Item not found');
    }

    // Delete item
    await this.databaseService.db.delete(listItems).where(eq(listItems.id, itemId));

    // Broadcast item deleted
    this.websocketService.broadcastToFamily(list.familyId, {
      type: 'list_updated',
      familyId: list.familyId,
      data: { action: 'item_deleted', listId, itemId },
    });

    return { success: true };
  }
}


