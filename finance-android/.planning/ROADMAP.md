# Roadmap: FinanceTracker

## Overview

FinanceTracker delivers a cross-platform expense tracking app in 10 phases. The journey starts with project scaffolding and authentication, builds up solo tracking (categories, expenses, history, charts), then layers on the family differentiator (groups, shared views), and finishes with offline sync and platform polish. Each phase delivers a coherent, verifiable capability that builds on the previous.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - Project scaffolding, monorepo structure, backend server, database, and cross-platform Flutter shell (completed 2026-03-15)
- [x] **Phase 2: Authentication** - User registration and persistent login sessions (completed 2026-03-15)
- [x] **Phase 3: Categories** - User-created expense categories with icons and colors (completed 2026-03-15)
- [ ] **Phase 4: Expense Entry** - Fast manual expense logging optimized for speed
- [ ] **Phase 5: Expense Management** - Edit and delete existing expenses
- [x] **Phase 6: History and Filtering** - Chronological expense history with date and category filters (completed 2026-03-16)
- [ ] **Phase 7: Visualization** - Spending charts and monthly summaries
- [ ] **Phase 8: Family Groups** - Create families and invite members
- [ ] **Phase 9: Family Views** - Shared expense feed and family summary dashboard
- [ ] **Phase 10: Offline and Platform Polish** - Offline support with sync, cross-platform refinements

## Phase Details

### Phase 1: Foundation
**Goal**: A running Flutter app shell and backend API server with database, deployable to all three platforms
**Depends on**: Nothing (first phase)
**Requirements**: PLAT-01
**Success Criteria** (what must be TRUE):
  1. Flutter app launches and renders a placeholder screen on Android, iOS, and web
  2. Backend API server starts and responds to a health check endpoint
  3. Database accepts connections and migrations run successfully
  4. Monorepo structure builds with a single command
**Plans**: 2 plans

Plans:
- [ ] 01-01-PLAN.md — Go backend server with PostgreSQL, Docker Compose, health endpoint, and dev tooling
- [ ] 01-02-PLAN.md — Flutter app shell with bottom navigation (3 tabs + FAB), flat design theme, and go_router

### Phase 2: Authentication
**Goal**: Users can create accounts and stay logged in across sessions
**Depends on**: Phase 1
**Requirements**: AUTH-01, AUTH-02
**Success Criteria** (what must be TRUE):
  1. User can create a new account with email and password
  2. User can log in with existing credentials and reach an authenticated home screen
  3. User remains logged in after closing and reopening the app
  4. User can log out and is returned to the login screen
**Plans**: 2 plans

Plans:
- [ ] 02-01-PLAN.md — Go backend auth: users table, JWT service, signup/login/refresh/logout endpoints, auth middleware
- [ ] 02-02-PLAN.md — Flutter auth UI: welcome/login/signup screens, secure storage, Dio interceptor, router guards, logout

### Phase 3: Categories
**Goal**: Users can build their own category system with visual identity (icons and colors)
**Depends on**: Phase 2
**Requirements**: CAT-01, CAT-02, CAT-03, CAT-04, CAT-05
**Success Criteria** (what must be TRUE):
  1. User can create a new category with a name
  2. User can assign an icon and color to a category
  3. User can edit a category name after creation
  4. User can delete a category
  5. Categories persist across sessions and display with their assigned icon and color
**Plans**: 2 plans

Plans:
- [ ] 03-01-PLAN.md — Go backend categories: migration, sqlc queries, CategoryHandler CRUD + reorder + bulk endpoints, unit tests
- [ ] 03-02-PLAN.md — Flutter categories UI: data layer, state management, list/form screens, icon/color pickers, drag-to-reorder, starter prompt, widget tests

### Phase 4: Expense Entry
**Goal**: Users can log expenses quickly with amount, category, optional note, and date
**Depends on**: Phase 3
**Requirements**: EXP-01, EXP-02, EXP-05
**Success Criteria** (what must be TRUE):
  1. User can log an expense with amount, category, optional note, and date
  2. A quick expense (amount + category only) can be entered in under 3 seconds
  3. Amounts display with proper locale formatting (e.g., $1,234.56) but are stored as integer cents
  4. Newly logged expenses appear immediately in the app
**Plans**: 2 plans

Plans:
- [ ] 04-01-PLAN.md — Go backend expenses: migration, sqlc queries, ExpenseDB interface, ExpenseHandler Create/List endpoints, unit tests
- [ ] 04-02-PLAN.md — Flutter expense form: model with cents conversion, repository, state management, full-screen form with amount/category/note/date, FAB wiring, widget tests

### Phase 5: Expense Management
**Goal**: Users can correct mistakes by editing or deleting any expense
**Depends on**: Phase 4
**Requirements**: EXP-03, EXP-04
**Success Criteria** (what must be TRUE):
  1. User can tap an expense and edit any field (amount, category, note, date)
  2. Edits are saved and reflected immediately in the expense list
  3. User can delete an expense and sees a confirmation dialog before deletion
**Plans**: 2 plans

Plans:
- [ ] 05-01-PLAN.md — Go backend: UpdateExpense/DeleteExpense sqlc queries, ExpenseDB interface extension, PUT/DELETE handlers with validation, unit tests
- [ ] 05-02-PLAN.md — Flutter edit/delete UI: repository+notifier update/delete methods, ExpenseFormScreen edit mode, HistoryScreen tap-to-edit + swipe-to-delete + category chips, widget tests

### Phase 6: History and Filtering
**Goal**: Users can review their spending history and find specific expenses
**Depends on**: Phase 5
**Requirements**: HIST-01, HIST-02, HIST-03
**Success Criteria** (what must be TRUE):
  1. User can view a scrollable list of all expenses sorted by date (newest first)
  2. User can filter expenses to a specific date range and see only matching results
  3. User can filter expenses by category and see only expenses in that category
  4. Filters can be combined (date range + category) and cleared
**Plans**: 2 plans

Plans:
- [ ] 06-01-PLAN.md — Go backend: GetExpensesByUserFiltered sqlc query with optional date_from/date_to/category_id params, List handler filter parsing, unit tests
- [ ] 06-02-PLAN.md — Flutter filter UI: FilterState provider, filter bar with date/category FilterChips, preset/category picker bottom sheets, widget tests

### Phase 7: Visualization
**Goal**: Users can see visual summaries of where their money goes
**Depends on**: Phase 6
**Requirements**: VIS-01, VIS-02, VIS-03
**Success Criteria** (what must be TRUE):
  1. User can view a pie chart showing spending breakdown by category
  2. User can view a bar chart showing spending over time (weekly or monthly)
  3. User can view a monthly summary showing total spent and per-category breakdown
  4. Charts update to reflect the current data (not stale)
**Plans**: TBD

Plans:
- [ ] 07-01: TBD
- [ ] 07-02: TBD

### Phase 8: Family Groups
**Goal**: Users can create family groups and invite others to join
**Depends on**: Phase 2
**Requirements**: FAM-01, FAM-02
**Success Criteria** (what must be TRUE):
  1. User can create a new family group with a name
  2. User can generate an invitation for another person to join their family
  3. An invited user can accept the invitation and become a family member
  4. Family membership persists and is visible in the user's profile or settings
**Plans**: TBD

Plans:
- [ ] 08-01: TBD
- [ ] 08-02: TBD

### Phase 9: Family Views
**Goal**: Family members can see each other's spending in a combined feed and summary
**Depends on**: Phase 8, Phase 6
**Requirements**: FAM-03, FAM-04
**Success Criteria** (what must be TRUE):
  1. Family members can view a combined expense feed showing who spent what
  2. Each expense in the family feed shows the name of the person who logged it
  3. Family members can view a summary dashboard with totals per person and per category
  4. Family views update when any member logs, edits, or deletes an expense
**Plans**: TBD

Plans:
- [ ] 09-01: TBD
- [ ] 09-02: TBD

### Phase 10: Offline and Platform Polish
**Goal**: The app works without connectivity and syncs when back online, with a polished experience on all platforms
**Depends on**: Phase 9
**Requirements**: PLAT-02
**Success Criteria** (what must be TRUE):
  1. User can log expenses while offline and they appear in the local list
  2. Offline expenses sync to the server when connectivity is restored
  3. App handles sync conflicts gracefully (no data loss, no duplicates)
  4. App is usable and visually consistent across Android, iOS, and web
**Plans**: TBD

Plans:
- [ ] 10-01: TBD
- [ ] 10-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10
Note: Phase 8 depends on Phase 2 (not Phase 7), so Phases 8-9 could theoretically parallel Phases 3-7 but are sequenced after for solo developer workflow.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 2/2 | Complete    | 2026-03-15 |
| 2. Authentication | 2/2 | Complete | 2026-03-15 |
| 3. Categories | 2/2 | Complete | 2026-03-15 |
| 4. Expense Entry | 1/2 | In Progress|  |
| 5. Expense Management | 0/2 | Not started | - |
| 6. History and Filtering | 2/2 | Complete   | 2026-03-16 |
| 7. Visualization | 0/? | Not started | - |
| 8. Family Groups | 0/? | Not started | - |
| 9. Family Views | 0/? | Not started | - |
| 10. Offline and Platform Polish | 0/? | Not started | - |
