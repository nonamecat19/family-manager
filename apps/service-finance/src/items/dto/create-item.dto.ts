import { IsOptional, IsString } from 'class-validator'

export class CreateItemDto {
  @IsString()
  name: string

  @IsString()
  @IsOptional()
  descriptions?: string
}
