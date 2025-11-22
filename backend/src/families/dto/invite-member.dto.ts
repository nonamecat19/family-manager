import { IsEmail } from 'class-validator';
import { ApiProperty } from '@nestjs/swagger';

export class InviteMemberDto {
  @ApiProperty({ example: 'member@example.com', description: 'Email of the member to invite' })
  @IsEmail()
  email: string;
}

