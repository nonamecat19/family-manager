import { IsString, IsUUID, MinLength, IsOptional } from 'class-validator';
import { ApiPropertyOptional } from '@nestjs/swagger';

export class UpdateBirthdayDto {
  @ApiPropertyOptional({ example: 'John', description: 'First name' })
  @IsOptional()
  @IsString()
  @MinLength(1)
  name?: string;

  @ApiPropertyOptional({ example: 'Doe', description: 'Last name' })
  @IsOptional()
  @IsString()
  surname?: string | null;

  @ApiPropertyOptional({ example: '1990-01-15', description: 'Date of birth (ISO date string)' })
  @IsOptional()
  @IsString()
  dateOfBirth?: string;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'User ID (UUID)' })
  @IsOptional()
  @IsUUID()
  userId?: string | null;
}

