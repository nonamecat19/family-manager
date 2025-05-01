import { IsNotEmpty, IsUUID } from 'class-validator';

export class DeleteCategoryDto {
  @IsNotEmpty()
  @IsUUID()
  id: string;
}
