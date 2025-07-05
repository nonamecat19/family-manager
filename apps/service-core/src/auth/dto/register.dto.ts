import {
  IsEmail,
  IsOptional,
  IsString,
  MaxLength,
  MinLength,
} from 'class-validator'

export class RegisterDto {
  @IsEmail()
  email: string

  @IsString()
  @MinLength(6)
  @MaxLength(255)
  password: string

  @IsOptional()
  @IsString()
  @MaxLength(50)
  name?: string

  @IsOptional()
  @IsString()
  @MaxLength(50)
  surname?: string
}
