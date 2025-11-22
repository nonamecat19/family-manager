import {
  Injectable,
  NotFoundException,
  ForbiddenException,
  BadRequestException,
} from '@nestjs/common';
import { DatabaseService } from '../database/database.service';
import { birthdays, familyMembers } from '../database/schema';
import { eq, and } from 'drizzle-orm';
import { CreateBirthdayDto } from './dto/create-birthday.dto';
import { UpdateBirthdayDto } from './dto/update-birthday.dto';
import { WebSocketService } from '../websocket/websocket.service';

@Injectable()
export class BirthdaysService {
  constructor(
    private databaseService: DatabaseService,
    private websocketService: WebSocketService,
  ) {}

  private calculateDaysUntil(dateOfBirth: Date): { daysUntil: number; nextBirthday: string } {
    const now = new Date();
    const currentYear = now.getFullYear();
    const today = new Date(currentYear, now.getMonth(), now.getDate());

    const birthDate = new Date(dateOfBirth);
    let nextBirthday = new Date(currentYear, birthDate.getMonth(), birthDate.getDate());

    // If birthday has already passed this year, use next year
    if (nextBirthday < today) {
      nextBirthday = new Date(currentYear + 1, birthDate.getMonth(), birthDate.getDate());
    }

    const daysUntil = Math.ceil((nextBirthday.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));

    return {
      daysUntil,
      nextBirthday: nextBirthday.toISOString().split('T')[0],
    };
  }

  async findAll(familyId: string, userId: string) {
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

    // Get all birthdays for the family
    const familyBirthdays = await this.databaseService.db.query.birthdays.findMany({
      where: eq(birthdays.familyId, familyId),
      with: {
        user: {
          columns: {
            id: true,
            name: true,
            email: true,
            avatar: true,
          },
        },
      },
    });

    // Sort by day (month and day, ignoring year)
    const sorted = familyBirthdays.sort((a, b) => {
      const dateA = new Date(a.dateOfBirth);
      const dateB = new Date(b.dateOfBirth);
      const monthA = dateA.getMonth();
      const monthB = dateB.getMonth();
      const dayA = dateA.getDate();
      const dayB = dateB.getDate();

      if (monthA !== monthB) {
        return monthA - monthB;
      }
      return dayA - dayB;
    });

    // Calculate days until for each birthday
    return sorted.map((birthday) => {
      const { daysUntil, nextBirthday } = this.calculateDaysUntil(new Date(birthday.dateOfBirth));
      return {
        ...birthday,
        daysUntil,
        nextBirthday,
      };
    });
  }

  async findUpcoming(familyId: string, limit: number, userId: string) {
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

    // Get all birthdays
    const allBirthdays = await this.databaseService.db.query.birthdays.findMany({
      where: eq(birthdays.familyId, familyId),
      with: {
        user: {
          columns: {
            id: true,
            name: true,
            email: true,
            avatar: true,
          },
        },
      },
    });

    // Calculate days until and filter upcoming
    const birthdaysWithDaysUntil = allBirthdays
      .map((birthday) => {
        const { daysUntil, nextBirthday } = this.calculateDaysUntil(new Date(birthday.dateOfBirth));
        return {
          ...birthday,
          daysUntil,
          nextBirthday,
        };
      })
      .sort((a, b) => a.daysUntil - b.daysUntil)
      .slice(0, limit);

    return birthdaysWithDaysUntil;
  }

  async findOne(id: string, userId: string) {
    const birthday = await this.databaseService.db.query.birthdays.findFirst({
      where: eq(birthdays.id, id),
      with: {
        family: true,
        user: {
          columns: {
            id: true,
            name: true,
            email: true,
            avatar: true,
          },
        },
      },
    });

    if (!birthday) {
      throw new NotFoundException('Birthday not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, birthday.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Calculate days until
    const { daysUntil, nextBirthday } = this.calculateDaysUntil(new Date(birthday.dateOfBirth));

    return {
      ...birthday,
      daysUntil,
      nextBirthday,
    };
  }

  async create(createBirthdayDto: CreateBirthdayDto, userId: string) {
    const { name, surname, dateOfBirth, familyId, userId: assignedUserId } = createBirthdayDto;

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

    // If userId is provided, verify it's a member of the family
    if (assignedUserId) {
      const assignedMembership = await this.databaseService.db.query.familyMembers.findFirst({
        where: and(
          eq(familyMembers.familyId, familyId),
          eq(familyMembers.userId, assignedUserId),
        ),
      });

      if (!assignedMembership) {
        throw new BadRequestException('User is not a member of the family');
      }
    }

    const [newBirthday] = await this.databaseService.db
      .insert(birthdays)
      .values({
        name,
        surname: surname || null,
        dateOfBirth,
        familyId,
        userId: assignedUserId || null,
      })
      .returning();

    // Broadcast birthday created
    this.websocketService.broadcastToFamily(familyId, {
      type: 'birthday_updated',
      familyId,
      data: { action: 'created', birthday: newBirthday },
    });

    return newBirthday;
  }

  async update(id: string, updateBirthdayDto: UpdateBirthdayDto, userId: string) {
    const birthday = await this.databaseService.db.query.birthdays.findFirst({
      where: eq(birthdays.id, id),
    });

    if (!birthday) {
      throw new NotFoundException('Birthday not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, birthday.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // If userId is being updated, verify it's a member of the family
    if (updateBirthdayDto.userId !== undefined && updateBirthdayDto.userId) {
      const assignedMembership = await this.databaseService.db.query.familyMembers.findFirst({
        where: and(
          eq(familyMembers.familyId, birthday.familyId),
          eq(familyMembers.userId, updateBirthdayDto.userId),
        ),
      });

      if (!assignedMembership) {
        throw new BadRequestException('User is not a member of the family');
      }
    }

    const updateData: Partial<typeof birthdays.$inferInsert> = {
      updatedAt: new Date(),
    };

    if (updateBirthdayDto.name !== undefined) {
      updateData.name = updateBirthdayDto.name;
    }
    if (updateBirthdayDto.surname !== undefined) {
      updateData.surname = updateBirthdayDto.surname;
    }
    if (updateBirthdayDto.dateOfBirth !== undefined) {
      updateData.dateOfBirth = updateBirthdayDto.dateOfBirth;
    }
    if (updateBirthdayDto.userId !== undefined) {
      updateData.userId = updateBirthdayDto.userId;
    }

    const [updatedBirthday] = await this.databaseService.db
      .update(birthdays)
      .set(updateData)
      .where(eq(birthdays.id, id))
      .returning();

    // Broadcast update
    this.websocketService.broadcastToFamily(birthday.familyId, {
      type: 'birthday_updated',
      familyId: birthday.familyId,
      data: { action: 'updated', birthday: updatedBirthday },
    });

    return updatedBirthday;
  }

  async remove(id: string, userId: string) {
    const birthday = await this.databaseService.db.query.birthdays.findFirst({
      where: eq(birthdays.id, id),
    });

    if (!birthday) {
      throw new NotFoundException('Birthday not found');
    }

    // Verify user is a member of the family
    const membership = await this.databaseService.db.query.familyMembers.findFirst({
      where: and(
        eq(familyMembers.familyId, birthday.familyId),
        eq(familyMembers.userId, userId),
      ),
    });

    if (!membership) {
      throw new ForbiddenException('Access denied');
    }

    // Delete birthday
    await this.databaseService.db.delete(birthdays).where(eq(birthdays.id, id));

    // Broadcast deletion
    this.websocketService.broadcastToFamily(birthday.familyId, {
      type: 'birthday_updated',
      familyId: birthday.familyId,
      data: { action: 'deleted', birthdayId: id },
    });

    return { success: true };
  }
}


