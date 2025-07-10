import { Transform } from 'class-transformer'
import { IsBoolean, IsNumber, IsOptional, IsPositive } from 'class-validator'

export class DeleteListDto {
  @IsNumber()
  @IsPositive()
  id: number

  @IsBoolean()
  @IsOptional()
  @Transform(({ value }) => value ?? false)
  withItems?: boolean
}
