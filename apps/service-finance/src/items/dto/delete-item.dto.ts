import { IsNumber, IsPositive } from 'class-validator'

export class DeleteItemDto {
  @IsNumber()
  @IsPositive()
  id: number
}
