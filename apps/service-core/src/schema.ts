import { relations } from 'drizzle-orm'
import {
  integer,
  pgTable,
  serial,
  timestamp,
  varchar,
} from 'drizzle-orm/pg-core'

export const workspaces = pgTable('workspaces', {
  id: serial('id').primaryKey(),
  name: varchar('name', { length: 255 }).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at')
    .$onUpdate(() => new Date())
    .defaultNow()
    .notNull(),
})

export const users = pgTable('users', {
  id: serial('id').primaryKey(),
  email: varchar('email', { length: 100 }).notNull().unique(),
  name: varchar('name', { length: 50 }),
  surname: varchar('surname', { length: 50 }),
  password: varchar('password', { length: 255 }),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at')
    .$onUpdate(() => new Date())
    .defaultNow()
    .notNull(),
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
