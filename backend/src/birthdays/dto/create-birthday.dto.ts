import { IsString, IsUUID, MinLength, IsOptional } from 'class-validator';
import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';

export class CreateBirthdayDto {
  @ApiProperty({ example: 'John', description: 'First name' })
  @IsString()
  @MinLength(1)
  name: string;

  @ApiPropertyOptional({ example: 'Doe', description: 'Last name' })
  @IsOptional()
  @IsString()
  surname?: string;

  @ApiProperty({ example: '1990-01-15', description: 'Date of birth (ISO date string)' })
  @IsString()
  dateOfBirth: string; // ISO date string

  @ApiProperty({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Family ID (UUID)' })
  @IsUUID()
  familyId: string;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'User ID (UUID) - optional if linking to a user' })
  @IsOptional()
  @IsUUID()
  userId?: string;
}

