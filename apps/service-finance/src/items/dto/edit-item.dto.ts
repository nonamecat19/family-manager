import { IsOptional, IsString } from 'class-validator'

export class EditItemDto {
  @IsString()
  @IsOptional()
  name?: string

  @IsString()
  @IsOptional()
  descriptions?: string
}
