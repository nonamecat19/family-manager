import { Injectable } from '@nestjs/common'
import { InjectDb } from '@repo/db'
import { eq } from 'drizzle-orm'
import { NodePgDatabase } from 'drizzle-orm/node-postgres'
import type * as schema from '../schema'
import { type NewUser, type User, users } from '../schema'

@Injectable()
export class UsersService {
  constructor(@InjectDb() private db: NodePgDatabase<typeof schema>) {}

  async create(newUser: NewUser): Promise<User> {
    const [user] = await this.db.insert(users).values(newUser).returning()
    return user
  }

  async findById(id: User['id']): Promise<User | undefined> {
    return this.db.query.users.findFirst({
      where: eq(users.id, id),
    })
  }

  async findByEmail(email: User['email']): Promise<User | undefined> {
    return this.db.query.users.findFirst({
      where: eq(users.email, email),
    })
  }
}
