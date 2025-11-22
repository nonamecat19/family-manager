import { IsString, MinLength, IsOptional } from 'class-validator';
import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';

export class CreateFamilyDto {
  @ApiProperty({ example: 'Smith Family', description: 'Family name' })
  @IsString()
  @MinLength(1)
  name: string;

  @ApiPropertyOptional({ example: '👨‍👩‍👧‍👦', description: 'Family icon' })
  @IsOptional()
  @IsString()
  icon?: string;

  @ApiPropertyOptional({ example: '#FF5733', description: 'Family color' })
  @IsOptional()
  @IsString()
  color?: string;
}

