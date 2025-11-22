import { IsString, IsUUID, MinLength, IsOptional } from 'class-validator';
import { ApiPropertyOptional } from '@nestjs/swagger';

export class UpdateFolderDto {
  @ApiPropertyOptional({ example: 'My Folder', description: 'Folder name' })
  @IsOptional()
  @IsString()
  @MinLength(1)
  name?: string;

  @ApiPropertyOptional({ example: '📁', description: 'Folder icon' })
  @IsOptional()
  @IsString()
  icon?: string;

  @ApiPropertyOptional({ example: '#FF5733', description: 'Folder color' })
  @IsOptional()
  @IsString()
  color?: string;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Parent folder ID (UUID)' })
  @IsOptional()
  @IsUUID()
  parentId?: string | null;
}

