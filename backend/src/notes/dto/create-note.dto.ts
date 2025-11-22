import { IsString, IsUUID, MinLength, IsOptional, IsEnum } from 'class-validator';
import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';

export enum NoteContentType {
  TEXT = 'text',
  LINK = 'link',
  COPY_TEXT = 'copy_text',
  FILE = 'file',
}

export class CreateNoteDto {
  @ApiProperty({ example: 'My Note', description: 'Note title' })
  @IsString()
  @MinLength(1)
  title: string;

  @ApiProperty({ example: 'text', enum: NoteContentType, description: 'Content type' })
  @IsEnum(NoteContentType)
  contentType: 'text' | 'link' | 'copy_text' | 'file';

  @ApiPropertyOptional({ example: 'Note content here', description: 'Note content' })
  @IsOptional()
  @IsString()
  content?: string;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Folder ID (UUID)' })
  @IsOptional()
  @IsUUID()
  folderId?: string;

  @ApiProperty({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Family ID (UUID)' })
  @IsUUID()
  familyId: string;
}

