import { systemFields } from '@repo/db'
import { relations } from 'drizzle-orm'
import { integer, pgTable, varchar } from 'drizzle-orm/pg-core'

export const workspaces = pgTable('workspaces', {
  ...systemFields,
  name: varchar('name', { length: 255 }).notNull(),
})

export const users = pgTable('users', {
  ...systemFields,
  email: varchar('email', { length: 100 }).notNull().unique(),
  name: varchar('name', { length: 50 }),
  surname: varchar('surname', { length: 50 }),
  password: varchar('password', { length: 255 }),
})

export const workspaceUsers = pgTable('workspace_users', {
  userId: integer('user_id')
    .references(() => users.id)
    .notNull(),
  workspaceId: integer('workspace_id')
    .references(() => workspaces.id)
    .notNull(),
})

export const workspacesRelations = relations(workspaces, ({ many }) => ({
  users: many(workspaceUsers),
}))

export const usersRelations = relations(users, ({ many }) => ({
  workspaces: many(workspaceUsers),
}))

export const workspaceUsersRelations = relations(workspaceUsers, ({ one }) => ({
  user: one(users, {
    fields: [workspaceUsers.userId],
    references: [users.id],
  }),
  workspace: one(workspaces, {
    fields: [workspaceUsers.workspaceId],
    references: [workspaces.id],
  }),
}))

export type Workspace = typeof workspaces.$inferSelect
export type NewWorkspace = typeof workspaces.$inferInsert
export type User = typeof users.$inferSelect
export type NewUser = typeof users.$inferInsert
