import { IsString, IsUUID, MinLength, IsOptional, IsEnum } from 'class-validator';
import { ApiPropertyOptional } from '@nestjs/swagger';
import { NoteContentType } from './create-note.dto';

export class UpdateNoteDto {
  @ApiPropertyOptional({ example: 'My Note', description: 'Note title' })
  @IsOptional()
  @IsString()
  @MinLength(1)
  title?: string;

  @ApiPropertyOptional({ example: 'text', enum: NoteContentType, description: 'Content type' })
  @IsOptional()
  @IsEnum(NoteContentType)
  contentType?: 'text' | 'link' | 'copy_text' | 'file';

  @ApiPropertyOptional({ example: 'Note content here', description: 'Note content' })
  @IsOptional()
  @IsString()
  content?: string;

  @ApiPropertyOptional({ example: '123e4567-e89b-12d3-a456-426614174000', description: 'Folder ID (UUID)' })
  @IsOptional()
  @IsUUID()
  folderId?: string | null;
}

