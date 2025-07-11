import { serial, timestamp } from 'drizzle-orm/pg-core'

export const systemFields = {
  id: serial('id').primaryKey(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at')
    .$onUpdate(() => new Date())
    .defaultNow()
    .notNull(),
} as const
