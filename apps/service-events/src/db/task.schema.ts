import { pgTable, serial, text } from 'drizzle-orm/pg-core'

export const task = pgTable('users', {
  id: serial('id').primaryKey(),
  name: text('name').notNull(),
  description: text('description'),
})
