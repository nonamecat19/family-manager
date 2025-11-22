import {
  pgTable,
  uuid,
  varchar,
  text,
  timestamp,
  boolean,
  date,
  time,
  integer,
  pgEnum,
} from 'drizzle-orm/pg-core';
import { relations } from 'drizzle-orm';

// Enums
export const familyRoleEnum = pgEnum('family_role', ['owner', 'member']);
export const folderTypeEnum = pgEnum('folder_type', ['list', 'note']);
export const noteContentTypeEnum = pgEnum('note_content_type', [
  'text',
  'link',
  'copy_text',
  'file',
]);

// Users table
export const users = pgTable('users', {
  id: uuid('id').defaultRandom().primaryKey(),
  email: varchar('email', { length: 255 }).notNull().unique(),
  passwordHash: text('password_hash').notNull(),
  name: varchar('name', { length: 255 }),
  avatar: text('avatar'),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// Families table
export const families = pgTable('families', {
  id: uuid('id').defaultRandom().primaryKey(),
  name: varchar('name', { length: 255 }).notNull(),
  icon: varchar('icon', { length: 50 }),
  color: varchar('color', { length: 7 }), // hex color
  createdBy: uuid('created_by').references(() => users.id).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// Family members table
export const familyMembers = pgTable('family_members', {
  id: uuid('id').defaultRandom().primaryKey(),
  userId: uuid('user_id').references(() => users.id).notNull(),
  familyId: uuid('family_id').references(() => families.id).notNull(),
  role: familyRoleEnum('role').notNull().default('member'),
  joinedAt: timestamp('joined_at').defaultNow().notNull(),
});

// Folders table
export const folders = pgTable('folders', {
  id: uuid('id').defaultRandom().primaryKey(),
  name: varchar('name', { length: 255 }).notNull(),
  icon: varchar('icon', { length: 50 }),
  color: varchar('color', { length: 7 }), // hex color
  familyId: uuid('family_id').references(() => families.id).notNull(),
  type: folderTypeEnum('type').notNull(), // 'list' or 'note'
  parentId: uuid('parent_id'), // for nested folders - self-reference handled in relations
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// Lists table
export const lists = pgTable('lists', {
  id: uuid('id').defaultRandom().primaryKey(),
  title: varchar('title', { length: 255 }).notNull(),
  description: text('description'),
  folderId: uuid('folder_id').references(() => folders.id),
  assignedTo: uuid('assigned_to').references(() => users.id),
  dueDate: date('due_date'),
  dueTime: time('due_time'),
  completed: boolean('completed').default(false).notNull(),
  familyId: uuid('family_id').references(() => families.id).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// List items table
export const listItems = pgTable('list_items', {
  id: uuid('id').defaultRandom().primaryKey(),
  listId: uuid('list_id').references(() => lists.id).notNull(),
  content: text('content').notNull(),
  order: integer('order').default(0).notNull(),
  completed: boolean('completed').default(false).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// Notes table
export const notes = pgTable('notes', {
  id: uuid('id').defaultRandom().primaryKey(),
  title: varchar('title', { length: 255 }).notNull(),
  contentType: noteContentTypeEnum('content_type').notNull().default('text'),
  content: text('content'),
  fileUrl: text('file_url'), // for file type notes
  folderId: uuid('folder_id').references(() => folders.id),
  familyId: uuid('family_id').references(() => families.id).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// Birthdays table
export const birthdays = pgTable('birthdays', {
  id: uuid('id').defaultRandom().primaryKey(),
  name: varchar('name', { length: 255 }).notNull(),
  surname: varchar('surname', { length: 255 }),
  dateOfBirth: date('date_of_birth').notNull(),
  familyId: uuid('family_id').references(() => families.id).notNull(),
  userId: uuid('user_id').references(() => users.id), // optional owner
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// Relations
export const usersRelations = relations(users, ({ many }) => ({
  families: many(familyMembers),
  createdFamilies: many(families),
  assignedLists: many(lists),
  birthdayEntries: many(birthdays),
}));

export const familiesRelations = relations(families, ({ one, many }) => ({
  creator: one(users, {
    fields: [families.createdBy],
    references: [users.id],
  }),
  members: many(familyMembers),
  folders: many(folders),
  lists: many(lists),
  notes: many(notes),
  birthdays: many(birthdays),
}));

export const familyMembersRelations = relations(familyMembers, ({ one }) => ({
  user: one(users, {
    fields: [familyMembers.userId],
    references: [users.id],
  }),
  family: one(families, {
    fields: [familyMembers.familyId],
    references: [families.id],
  }),
}));

export const foldersRelations = relations(folders, ({ one, many }) => ({
  family: one(families, {
    fields: [folders.familyId],
    references: [families.id],
  }),
  parent: one(folders, {
    fields: [folders.parentId],
    references: [folders.id],
    relationName: 'parent',
  }),
  children: many(folders, {
    relationName: 'parent',
  }),
  lists: many(lists),
  notes: many(notes),
}));

export const listsRelations = relations(lists, ({ one, many }) => ({
  family: one(families, {
    fields: [lists.familyId],
    references: [families.id],
  }),
  folder: one(folders, {
    fields: [lists.folderId],
    references: [folders.id],
  }),
  assignedUser: one(users, {
    fields: [lists.assignedTo],
    references: [users.id],
  }),
  items: many(listItems),
}));

export const listItemsRelations = relations(listItems, ({ one }) => ({
  list: one(lists, {
    fields: [listItems.listId],
    references: [lists.id],
  }),
}));

export const notesRelations = relations(notes, ({ one }) => ({
  family: one(families, {
    fields: [notes.familyId],
    references: [families.id],
  }),
  folder: one(folders, {
    fields: [notes.folderId],
    references: [folders.id],
  }),
}));

export const birthdaysRelations = relations(birthdays, ({ one }) => ({
  family: one(families, {
    fields: [birthdays.familyId],
    references: [families.id],
  }),
  user: one(users, {
    fields: [birthdays.userId],
    references: [users.id],
  }),
}));

