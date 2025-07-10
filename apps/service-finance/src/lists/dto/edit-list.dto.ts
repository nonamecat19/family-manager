import { IsOptional, IsString } from 'class-validator'

export class EditListDto {
  @IsString()
  @IsOptional()
  name?: string

  @IsString()
  @IsOptional()
  descriptions?: string
}
