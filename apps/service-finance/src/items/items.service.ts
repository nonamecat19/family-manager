import { Injectable } from '@nestjs/common'
import { InjectDb } from '@repo/db'
import { eq } from 'drizzle-orm'
import { NodePgDatabase } from 'drizzle-orm/node-postgres'
import * as schema from '../schema'
import { items } from '../schema'
import { CreateItemDto } from './dto'

@Injectable()
export class ItemsService {
  constructor(@InjectDb() private db: NodePgDatabase<typeof schema>) {}

  async getAll() {
    return this.db.query.items.findMany()
  }

  async createOne(newItem: CreateItemDto) {
    const [item] = await this.db.insert(items).values(newItem).returning()
    return item
  }

  async deleteOne(id: number) {
    const [item] = await this.db
      .delete(items)
      .where(eq(items.id, id))
      .returning()
    return item
  }
}
