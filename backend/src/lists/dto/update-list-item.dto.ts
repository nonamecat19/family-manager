import { IsString, MinLength, IsOptional, IsNumber, IsBoolean } from 'class-validator';
import { ApiPropertyOptional } from '@nestjs/swagger';

export class UpdateListItemDto {
  @ApiPropertyOptional({ example: 'Buy milk', description: 'Item content' })
  @IsOptional()
  @IsString()
  @MinLength(1)
  content?: string;

  @ApiPropertyOptional({ example: 1, description: 'Item order/index' })
  @IsOptional()
  @IsNumber()
  order?: number;

  @ApiPropertyOptional({ example: false, description: 'Completion status' })
  @IsOptional()
  @IsBoolean()
  completed?: boolean;
}

