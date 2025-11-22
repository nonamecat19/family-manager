import { IsString, IsUUID, MinLength, IsOptional, IsEnum } from 'class-validator';
import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';

export enum FolderType {
  LIST = 'list',
  NOTE = 'note',
}

export class CreateFolderDto {
  @ApiProperty({ example: 'My Folder', description: 'Folder name' })
  @IsString()
  @MinLength(1)
  name: string;

  @ApiPropertyOptional({ example: '📁', description: 'Folder icon' })
  @IsOptional()
  @IsString()
  icon?: string;

  @ApiPropertyOptional({ example: '#FF5733', description: 'Folder color' })
  @IsOptional()
  @IsString()
  color?: string;

  @ApiProperty({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Family ID (UUID)' })
  @IsUUID()
  familyId: string;

  @ApiProperty({ example: 'note', enum: FolderType, description: 'Folder type' })
  @IsEnum(FolderType)
  type: 'list' | 'note';

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Parent folder ID (UUID)' })
  @IsOptional()
  @IsUUID()
  parentId?: string;
}

