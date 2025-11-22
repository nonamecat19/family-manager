import { IsString, MinLength, IsOptional, IsNumber } from 'class-validator';
import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';

export class CreateListItemDto {
  @ApiProperty({ example: 'Buy milk', description: 'Item content' })
  @IsString()
  @MinLength(1)
  content: string;

  @ApiPropertyOptional({ example: 1, description: 'Item order/index' })
  @IsOptional()
  @IsNumber()
  order?: number;
}

