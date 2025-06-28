import {integer, pgTable, serial, timestamp, varchar} from 'drizzle-orm/pg-core';
import {relations} from 'drizzle-orm';

export const workspaces = pgTable('workspaces', {
    id: serial('id').primaryKey(),
    name: varchar('name', {length: 255}).notNull(),
    createdAt: timestamp('created_at').defaultNow().notNull(),
    updatedAt: timestamp('updated_at').$onUpdate(() => new Date()).defaultNow().notNull(),
});

export const users = pgTable('users', {
    id: serial('id').primaryKey(),
    email: varchar('email', {length: 100}).notNull().unique(),
    name: varchar('name', {length: 50}).notNull(),
    surname: varchar('name', {length: 50}).notNull(),
    password: varchar('password', {length: 255}).notNull(),
    workspaceId: integer('workspace_id').notNull().references(() => workspaces.id),
    createdAt: timestamp('created_at').defaultNow().notNull(),
    updatedAt: timestamp('updated_at').$onUpdate(() => new Date()).defaultNow().notNull(),
});

export const workspacesRelations = relations(workspaces, ({many}) => ({
    users: many(users),
}));

export const usersRelations = relations(users, ({one}) => ({
    workspace: one(workspaces, {
        fields: [users.workspaceId],
        references: [workspaces.id],
    }),
}));

export type Workspace = typeof workspaces.$inferSelect;
export type NewWorkspace = typeof workspaces.$inferInsert;
export type User = typeof users.$inferSelect;
export type NewUser = typeof users.$inferInsert;