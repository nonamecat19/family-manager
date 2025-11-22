import { IsEnum } from 'class-validator';
import { ApiProperty } from '@nestjs/swagger';

export enum FamilyRole {
  OWNER = 'owner',
  MEMBER = 'member',
}

export class UpdateMemberRoleDto {
  @ApiProperty({ example: 'member', enum: FamilyRole, description: 'Member role' })
  @IsEnum(FamilyRole)
  role: 'owner' | 'member';
}

