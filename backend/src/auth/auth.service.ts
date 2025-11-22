import {
  Injectable,
  UnauthorizedException,
  BadRequestException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { ConfigService } from '@nestjs/config';
import { DatabaseService } from '../database/database.service';
import { users, families, familyMembers } from '../database/schema';
import { eq } from 'drizzle-orm';
import { PasswordUtil } from '../common/utils/password.util';
import { RegisterDto } from './dto/register.dto';
import { LoginDto } from './dto/login.dto';

@Injectable()
export class AuthService {
  constructor(
    private databaseService: DatabaseService,
    private jwtService: JwtService,
    private configService: ConfigService,
  ) {}

  async register(registerDto: RegisterDto) {
    const { email, password, name } = registerDto;

    // Check if user exists
    let existingUser;
    try {
      existingUser = await this.databaseService.db.query.users.findFirst({
        where: eq(users.email, email),
      });
    } catch (error) {
      console.error('Database error when checking existing user:', error);
      throw new BadRequestException(
        `Database error: ${error instanceof Error ? error.message : 'Unknown error'}`,
      );
    }

    if (existingUser) {
      throw new BadRequestException('User already exists');
    }

    // Hash password
    const passwordHash = await PasswordUtil.hash(password);

    // Create user
    let newUser;
    try {
      [newUser] = await this.databaseService.db
        .insert(users)
        .values({
          email,
          passwordHash,
          name: name || null,
        })
        .returning();
    } catch (error) {
      console.error('Database error when creating user:', error);
      throw new BadRequestException(
        `Failed to create user: ${error instanceof Error ? error.message : 'Unknown error'}`,
      );
    }

    // Create private family/workspace for the user
    let privateFamily;
    try {
      [privateFamily] = await this.databaseService.db
        .insert(families)
        .values({
          name: `${name || 'My'} Workspace`,
          createdBy: newUser.id,
        })
        .returning();
    } catch (error) {
      console.error('Database error when creating family:', error);
      throw new BadRequestException(
        `Failed to create family: ${error instanceof Error ? error.message : 'Unknown error'}`,
      );
    }

    // Add user as owner of their private family
    try {
      await this.databaseService.db.insert(familyMembers).values({
        userId: newUser.id,
        familyId: privateFamily.id,
        role: 'owner',
      });
    } catch (error) {
      console.error('Database error when adding family member:', error);
      throw new BadRequestException(
        `Failed to add family member: ${error instanceof Error ? error.message : 'Unknown error'}`,
      );
    }

    // Generate JWT token
    const token = await this.generateToken(newUser.id, newUser.email);

    return {
      token,
      user: {
        id: newUser.id,
        email: newUser.email,
        name: newUser.name,
      },
      defaultFamily: {
        id: privateFamily.id,
        name: privateFamily.name,
      },
    };
  }

  async login(loginDto: LoginDto) {
    const { email, password } = loginDto;

    // Find user
    const user = await this.databaseService.db.query.users.findFirst({
      where: eq(users.email, email),
    });

    if (!user) {
      throw new UnauthorizedException('Invalid credentials');
    }

    // Verify password
    const isValid = await PasswordUtil.compare(password, user.passwordHash);

    if (!isValid) {
      throw new UnauthorizedException('Invalid credentials');
    }

    // Generate JWT token
    const token = await this.generateToken(user.id, user.email);

    return {
      token,
      user: {
        id: user.id,
        email: user.email,
        name: user.name,
      },
    };
  }

  async refreshToken(userId: string, email: string) {
    const token = await this.generateToken(userId, email);
    return { token };
  }

  async getCurrentUser(userId: string) {
    const user = await this.databaseService.db.query.users.findFirst({
      where: eq(users.id, userId),
    });

    if (!user) {
      throw new UnauthorizedException('User not found');
    }

    return {
      id: user.id,
      email: user.email,
      name: user.name,
      avatar: user.avatar,
    };
  }

  private async generateToken(userId: string, email: string): Promise<string> {
    const payload = { userId, email };
    const secret = this.configService.get<string>('JWT_SECRET') || 'your-secret-key';
    return this.jwtService.signAsync(payload, { secret });
  }
}


