import { Injectable } from '@nestjs/common'
import { InjectDb } from '@repo/db'
import { eq } from 'drizzle-orm'
import { NodePgDatabase } from 'drizzle-orm/node-postgres'
import * as schema from '../schema'
import { items, List, lists, NewList } from '../schema'

@Injectable()
export class ListsService {
  constructor(@InjectDb() private db: NodePgDatabase<typeof schema>) {}

  async getAll() {
    return this.db.query.lists.findMany({
      with: {
        items: true,
      },
    })
  }

  async getById(id: List['id']) {
    return this.db.query.lists.findFirst({
      with: {
        items: true,
      },
      where: eq(lists.id, id),
    })
  }

  async createOne(newList: NewList) {
    const [list] = await this.db.insert(lists).values(newList).returning()
    return list
  }

  async deleteOne(id: List['id'], withItems: boolean = false) {
    const [item] = await this.db
      .delete(lists)
      .where(eq(lists.id, id))
      .returning()

    if (withItems) {
      await this.db.delete(items).where(eq(items.id, id))
    }

    return item
  }
}
