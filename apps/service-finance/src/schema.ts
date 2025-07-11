import { systemFields } from '@repo/db'
import { relations } from 'drizzle-orm'
import { boolean, integer, pgTable, text } from 'drizzle-orm/pg-core'

export const lists = pgTable('lists', {
  ...systemFields,
  name: text('name').notNull(),
  description: text('description'),
})

export const items = pgTable('items', {
  ...systemFields,
  name: text('name').notNull(),
  description: text('description'),
  quantity: integer('quantity').default(1).notNull(),
  isCompleted: boolean('is_completed').default(false).notNull(),
  listId: integer('list_id').references(() => lists.id, {
    onDelete: 'set null',
  }),
})

export const listsRelations = relations(lists, ({ many }) => ({
  items: many(items),
}))

export const itemsRelations = relations(items, ({ one }) => ({
  list: one(lists, {
    fields: [items.listId],
    references: [lists.id],
  }),
}))

export type List = typeof lists.$inferSelect
export type NewList = typeof lists.$inferInsert
export type Item = typeof items.$inferSelect
export type NewItem = typeof items.$inferInsert
