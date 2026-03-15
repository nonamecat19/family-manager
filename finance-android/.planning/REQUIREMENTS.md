# Requirements: FinanceTracker

**Defined:** 2026-03-15
**Core Value:** Users can quickly log expenses, categorize them their way, and understand where their money goes — individually or as a family.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Authentication

- [x] **AUTH-01**: User can create account with email and password
- [x] **AUTH-02**: User can log in and stay logged in across sessions

### Categories

- [x] **CAT-01**: User can create custom expense categories
- [x] **CAT-02**: User can edit category name
- [x] **CAT-03**: User can delete categories
- [x] **CAT-04**: User can assign an icon to each category
- [x] **CAT-05**: User can assign a color to each category

### Expenses

- [x] **EXP-01**: User can log an expense with amount, category, optional note, and date
- [ ] **EXP-02**: Expense entry is optimized for speed (under 3 seconds for quick entry)
- [ ] **EXP-03**: User can edit any field of an existing expense
- [ ] **EXP-04**: User can delete an expense with confirmation
- [x] **EXP-05**: Amounts stored as integer cents, displayed with locale formatting

### History

- [ ] **HIST-01**: User can view expense history sorted by date (newest first)
- [ ] **HIST-02**: User can filter expenses by date range
- [ ] **HIST-03**: User can filter expenses by category

### Visualization

- [ ] **VIS-01**: User can view pie chart of spending by category
- [ ] **VIS-02**: User can view bar chart of spending over time (weekly/monthly)
- [ ] **VIS-03**: User can view monthly summary (total spent, category breakdown)

### Family

- [ ] **FAM-01**: User can create a family group
- [ ] **FAM-02**: User can invite others to join their family
- [ ] **FAM-03**: Family members can view combined expense feed showing who spent what
- [ ] **FAM-04**: Family members can view summary dashboard with per-person and per-category totals

### Platform

- [x] **PLAT-01**: App runs on Android, iOS, and web from single Flutter codebase
- [ ] **PLAT-02**: App works offline and syncs when connectivity is restored

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Authentication

- **AUTH-V2-01**: User can log in with Google OAuth
- **AUTH-V2-02**: User can log in with Apple Sign-In
- **AUTH-V2-03**: User can reset password via email link

### Features

- **FEAT-V2-01**: User can search expenses by description text
- **FEAT-V2-02**: User can export expense data to CSV
- **FEAT-V2-03**: User receives notifications/reminders to log expenses
- **FEAT-V2-04**: User can set up recurring expenses

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Bank sync / automatic import | Massive complexity (Plaid, deduplication). Manual simplicity is the feature. |
| Receipt scanning / OCR | ML pipeline complexity, low accuracy without major investment |
| Budget limits and alerts | Different product mindset — tracking first, budgeting later |
| Debt/loan tracking | Different domain, adds schema complexity |
| Investment tracking | Completely different domain requiring market data APIs |
| Multi-currency conversion | Exchange rate APIs and conversion logic — single currency per user for v1 |
| AI-powered categorization | Requires ML pipeline — manual category selection is the design choice |
| Social features / leaderboards | Gamifying personal finance creates bad incentives |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| PLAT-01 | Phase 1: Foundation | Complete |
| AUTH-01 | Phase 2: Authentication | Complete |
| AUTH-02 | Phase 2: Authentication | Complete |
| CAT-01 | Phase 3: Categories | API Complete (Plan 1) |
| CAT-02 | Phase 3: Categories | API Complete (Plan 1) |
| CAT-03 | Phase 3: Categories | API Complete (Plan 1) |
| CAT-04 | Phase 3: Categories | API Complete (Plan 1) |
| CAT-05 | Phase 3: Categories | API Complete (Plan 1) |
| EXP-01 | Phase 4: Expense Entry | Complete |
| EXP-02 | Phase 4: Expense Entry | Pending |
| EXP-05 | Phase 4: Expense Entry | Complete |
| EXP-03 | Phase 5: Expense Management | Pending |
| EXP-04 | Phase 5: Expense Management | Pending |
| HIST-01 | Phase 6: History and Filtering | Pending |
| HIST-02 | Phase 6: History and Filtering | Pending |
| HIST-03 | Phase 6: History and Filtering | Pending |
| VIS-01 | Phase 7: Visualization | Pending |
| VIS-02 | Phase 7: Visualization | Pending |
| VIS-03 | Phase 7: Visualization | Pending |
| FAM-01 | Phase 8: Family Groups | Pending |
| FAM-02 | Phase 8: Family Groups | Pending |
| FAM-03 | Phase 9: Family Views | Pending |
| FAM-04 | Phase 9: Family Views | Pending |
| PLAT-02 | Phase 10: Offline and Platform Polish | Pending |

**Coverage:**
- v1 requirements: 24 total
- Mapped to phases: 24
- Unmapped: 0

---
*Requirements defined: 2026-03-15*
*Last updated: 2026-03-15 after roadmap creation*
