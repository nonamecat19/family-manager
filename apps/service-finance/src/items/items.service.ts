import { Injectable } from '@nestjs/common'
import { InjectDb } from '@repo/db'
import { eq } from 'drizzle-orm'
import { NodePgDatabase } from 'drizzle-orm/node-postgres'
import * as schema from '../schema'
import { Item, items, NewItem } from '../schema'

@Injectable()
export class ItemsService {
  constructor(@InjectDb() private db: NodePgDatabase<typeof schema>) {}

  async getAll() {
    return this.db.query.items.findMany()
  }

  async createOne(newItem: NewItem) {
    const [item] = await this.db.insert(items).values(newItem).returning()
    return item
  }

  async deleteOne(id: Item['id']) {
    const [item] = await this.db
      .delete(items)
      .where(eq(items.id, id))
      .returning()
    return item
  }
}
