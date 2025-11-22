import { IsString, IsUUID, MinLength, IsOptional, IsBoolean } from 'class-validator';
import { ApiPropertyOptional } from '@nestjs/swagger';

export class UpdateListDto {
  @ApiPropertyOptional({ example: 'Shopping List', description: 'List title' })
  @IsOptional()
  @IsString()
  @MinLength(1)
  title?: string;

  @ApiPropertyOptional({ example: 'Grocery shopping list', description: 'List description' })
  @IsOptional()
  @IsString()
  description?: string;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Folder ID (UUID)' })
  @IsOptional()
  @IsUUID()
  folderId?: string | null;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Assigned user ID (UUID)' })
  @IsOptional()
  @IsUUID()
  assignedTo?: string | null;

  @ApiPropertyOptional({ example: '2024-12-25', description: 'Due date (ISO date string)' })
  @IsOptional()
  @IsString()
  dueDate?: string | null;

  @ApiPropertyOptional({ example: '14:00', description: 'Due time' })
  @IsOptional()
  @IsString()
  dueTime?: string | null;

  @ApiPropertyOptional({ example: false, description: 'Completion status' })
  @IsOptional()
  @IsBoolean()
  completed?: boolean;
}

