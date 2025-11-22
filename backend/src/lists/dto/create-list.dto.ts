import { IsString, IsUUID, MinLength, IsOptional } from 'class-validator';
import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';

export class CreateListDto {
  @ApiProperty({ example: 'Shopping List', description: 'List title' })
  @IsString()
  @MinLength(1)
  title: string;

  @ApiPropertyOptional({ example: 'Grocery shopping list', description: 'List description' })
  @IsOptional()
  @IsString()
  description?: string;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Folder ID (UUID)' })
  @IsOptional()
  @IsUUID()
  folderId?: string;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Assigned user ID (UUID)' })
  @IsOptional()
  @IsUUID()
  assignedTo?: string;

  @ApiPropertyOptional({ example: '2024-12-25', description: 'Due date (ISO date string)' })
  @IsOptional()
  @IsString()
  dueDate?: string;

  @ApiPropertyOptional({ example: '14:00', description: 'Due time' })
  @IsOptional()
  @IsString()
  dueTime?: string;

  @ApiProperty({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Family ID (UUID)' })
  @IsUUID()
  familyId: string;
}

