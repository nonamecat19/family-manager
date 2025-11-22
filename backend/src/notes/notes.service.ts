import {
  Injectable,
  NotFoundException,
  ForbiddenException,
  BadRequestException,
  InternalServerErrorException,
} from '@nestjs/common';
import { DatabaseService } from '../database/database.service';
import { notes, familyMembers, folders } from '../database/schema';
import { eq, and, desc } from 'drizzle-orm';
import { CreateNoteDto } from './dto/create-note.dto';
import { UpdateNoteDto } from './dto/update-note.dto';
import { WebSocketService } from '../websocket/websocket.service';
import { put } from '@vercel/blob';
import { ConfigService } from '@nestjs/config';

@Injectable()
export class NotesService {
  constructor(
    private databaseService: DatabaseService,
    private websocketService: WebSocketService,
    private configService: ConfigService,
  ) {}

  async findAll(familyId: string, folderId: string | undefined, userId: string) {
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
    const conditions = [eq(notes.familyId, familyId)];

    if (folderId) {
      conditions.push(eq(notes.folderId, folderId));
    }

    const userNotes = await this.databaseService.db.query.notes.findMany({
      where: and(...conditions),
      with: {
        folder: true,
      },
      orderBy: (notes, { desc }) => [desc(notes.createdAt)],
    });

    return userNotes;
  }

  async findOne(id: string, userId: string) {
    const note = await this.databaseService.db.query.notes.findFirst({
      where: eq(notes.id, id),
      with: {
        family: true,
        folder: true,
      },
    });

    if (!note) {
      throw new NotFoundException('Note not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, note.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    return note;
  }

  async create(createNoteDto: CreateNoteDto, userId: string) {
    const { title, contentType, content, folderId, familyId } = createNoteDto;

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

      if (!folder || folder.familyId !== familyId || folder.type !== 'note') {
        throw new BadRequestException('Invalid folder');
      }
    }

    const [newNote] = await this.databaseService.db
      .insert(notes)
      .values({
        title,
        contentType,
        content: content || null,
        fileUrl: null, // Will be set when file is uploaded
        folderId: folderId || null,
        familyId,
      })
      .returning();

    // Broadcast note created
    this.websocketService.broadcastToFamily(familyId, {
      type: 'note_updated',
      familyId,
      data: { action: 'created', note: newNote },
    });

    return newNote;
  }

  async update(id: string, updateNoteDto: UpdateNoteDto, userId: string) {
    const note = await this.databaseService.db.query.notes.findFirst({
      where: eq(notes.id, id),
    });

    if (!note) {
      throw new NotFoundException('Note not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, note.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Validate folder if being updated
    if (updateNoteDto.folderId !== undefined && updateNoteDto.folderId) {
      const folder = await this.databaseService.db.query.folders.findFirst({
        where: eq(folders.id, updateNoteDto.folderId),
      });

      if (!folder || folder.familyId !== note.familyId || folder.type !== 'note') {
        throw new BadRequestException('Invalid folder');
      }
    }

    const updateData: Partial<typeof notes.$inferInsert> = {
      updatedAt: new Date(),
    };

    if (updateNoteDto.title !== undefined) {
      updateData.title = updateNoteDto.title;
    }
    if (updateNoteDto.contentType !== undefined) {
      updateData.contentType = updateNoteDto.contentType;
    }
    if (updateNoteDto.content !== undefined) {
      updateData.content = updateNoteDto.content;
    }
    if (updateNoteDto.folderId !== undefined) {
      updateData.folderId = updateNoteDto.folderId;
    }

    const [updatedNote] = await this.databaseService.db
      .update(notes)
      .set(updateData)
      .where(eq(notes.id, id))
      .returning();

    // Broadcast update
    this.websocketService.broadcastToFamily(note.familyId, {
      type: 'note_updated',
      familyId: note.familyId,
      data: { action: 'updated', note: updatedNote },
    });

    return updatedNote;
  }

  async remove(id: string, userId: string) {
    const note = await this.databaseService.db.query.notes.findFirst({
      where: eq(notes.id, id),
    });

    if (!note) {
      throw new NotFoundException('Note not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, note.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Delete note
    await this.databaseService.db.delete(notes).where(eq(notes.id, id));

    // Broadcast deletion
    this.websocketService.broadcastToFamily(note.familyId, {
      type: 'note_updated',
      familyId: note.familyId,
      data: { action: 'deleted', noteId: id },
    });

    return { success: true };
  }

  async uploadFile(id: string, file: Buffer, filename: string, userId: string) {
    const note = await this.databaseService.db.query.notes.findFirst({
      where: eq(notes.id, id),
    });

    if (!note) {
      throw new NotFoundException('Note not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, note.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Verify note is of file type
    if (note.contentType !== 'file') {
      throw new BadRequestException('Note must be of type file');
    }

    // Upload to Vercel Blob
    const token = this.configService.get<string>('BLOB_READ_WRITE_TOKEN');
    if (!token) {
      throw new InternalServerErrorException('Blob storage not configured');
    }

    try {
      const blob = await put(filename, file, {
        access: 'public',
        token,
      });

      // Update note with file URL
      const [updatedNote] = await this.databaseService.db
        .update(notes)
        .set({
          fileUrl: blob.url,
          updatedAt: new Date(),
        })
        .where(eq(notes.id, id))
        .returning();

      // Broadcast update
      this.websocketService.broadcastToFamily(note.familyId, {
        type: 'note_updated',
        familyId: note.familyId,
        data: { action: 'file_uploaded', note: updatedNote },
      });

      return { url: blob.url, note: updatedNote };
    } catch (uploadError) {
      console.error('Upload error:', uploadError);
      throw new InternalServerErrorException('Failed to upload file');
    }
  }
}


