# Personal Finance Manager - Backend

A personal finance management service built with Golang, GraphQL, PostgreSQL, and SQLC.

## Architecture

This project follows Clean Architecture principles with the following layers:

- **Domain Layer** (`internal/domain`): Core business entities and repository interfaces
- **Use Case Layer** (`internal/usecase`): Application-specific business logic
- **Repository Layer** (`internal/repository`): Database access implementations using SQLC
- **Handler Layer** (`internal/handler`): GraphQL resolvers and API handlers
- **Infrastructure Layer** (`internal/infrastructure`): Database, configuration, authentication, and scheduling

## Features

- Multi-currency wallet management (cash, credit cards, crypto)
- Transaction tracking (income, expenses, transfers)
- Recurring payments and income
- Investment tracking (crypto staking, etc.)
- Automated recurring payment processing via cron scheduler
- JWT-based authentication (structure ready)

## Prerequisites

- Go 1.21 or later
- PostgreSQL 12 or later
- SQLC (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- golang-migrate (`go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`)
- gqlgen (`go install github.com/99designs/gqlgen@latest`)

## Setup

1. **Clone the repository** (if applicable)

2. **Install dependencies**:
   ```bash
   make deps
   # or
   go mod download
   ```

3. **Start PostgreSQL** (using Docker Compose):
   ```bash
   docker-compose up -d postgres
   ```
   This will start PostgreSQL on `localhost:5432` with default credentials:
   - User: `postgres`
   - Password: `postgres`
   - Database: `finance_manager`

4. **Set up environment variables** (optional - defaults work for local dev):
   Create a `.env` file in the `backend/` directory:
   ```env
   DATABASE_URL=postgres://postgres:postgres@localhost:5432/finance_manager?sslmode=disable
   JWT_SECRET=your-secret-key-change-in-production
   PORT=8080
   ENVIRONMENT=development
   ```
   Note: The default DATABASE_URL already matches the docker-compose PostgreSQL setup.

5. **Run database migrations**:
   ```bash
   make migrate-up
   # or
   migrate -path migrations -database "$DATABASE_URL" up
   ```

6. **Generate code**:
   ```bash
   # Generate SQLC code (if you modified queries)
   make sqlc
   
   # Generate GraphQL code
   make generate
   # or
   go run github.com/99designs/gqlgen generate
   ```

7. **Build and run**:
   ```bash
   make build
   ./bin/server
   
   # or run directly
   make run
   ```

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── domain/
│   │   ├── entity/          # Domain entities
│   │   └── repository/      # Repository interfaces
│   ├── usecase/             # Business logic
│   ├── repository/
│   │   └── postgres/        # SQLC generated code + implementations
│   ├── handler/
│   │   └── graphql/         # GraphQL schema and resolvers
│   └── infrastructure/
│       ├── config/          # Configuration management
│       ├── database/        # Database connection
│       ├── auth/            # JWT and password hashing
│       └── scheduler/       # Cron job scheduler
├── migrations/              # Database migrations
├── sqlc.yaml               # SQLC configuration
├── gqlgen.yml              # GraphQL code generation config
└── Makefile                # Build commands
```

## Database Schema

The application uses PostgreSQL with the following main tables:

- `users`: User accounts
- `wallets`: Financial accounts (cash, credit cards, crypto)
- `transactions`: Income, expenses, and transfers
- `recurring`: Recurring payments and income
- `investments`: Investments such as crypto staking

See `migrations/001_initial_schema.up.sql` for the complete schema.

## GraphQL API

The GraphQL API is defined in `internal/handler/graphql/schema.graphqls`.

### Queries

- `wallets(userId: ID!)`: Get all wallets for a user
- `transactions(walletId: ID!, from: String, to: String)`: Get transactions for a wallet
- `investments(userId: ID!)`: Get all investments for a user
- `recurringPayments(userId: ID!)`: Get all recurring payments for a user

### Mutations

- `createWallet`: Create a new wallet
- `updateWallet`: Update wallet details
- `deleteWallet`: Delete a wallet
- `createTransaction`: Create a transaction
- `createRecurring`: Create a recurring payment/income
- `createInvestment`: Create an investment

## Scheduler

The application includes a cron scheduler that runs:

- **Every hour**: Processes due recurring payments and creates transactions
- **Daily at midnight**: Processes investment rewards (to be implemented)

## Development

### Running Tests

```bash
make test
```

### Code Generation

After modifying SQL queries or GraphQL schema:

```bash
# Generate SQLC code
make sqlc

# Generate GraphQL resolvers
make generate
```

### Database Migrations

```bash
# Create a new migration
migrate create -ext sql -dir migrations -seq migration_name

# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down
```

## Important Notes

### GraphQL Code Generation

Before running the application, you need to generate GraphQL code:

```bash
go get github.com/99designs/gqlgen
go run github.com/99designs/gqlgen generate
```

This will generate the resolver stubs in `internal/handler/graphql/resolver/` that need to be implemented.

### Numeric Conversion

The current implementation uses a simplified approach for converting between `float64` and `pgtype.Numeric`. For production use, consider using a decimal library like `shopspring/decimal` for more accurate financial calculations.

## Future Enhancements

- [ ] Complete GraphQL resolver implementation (resolvers need to be generated and implemented)
- [ ] User authentication endpoints (login/register)
- [ ] JWT middleware for GraphQL
- [ ] Currency conversion API integration
- [ ] Bank/crypto API integrations
- [ ] Push notifications
- [ ] Advanced analytics and reporting
- [ ] Unit and integration tests
- [ ] Use decimal library for precise financial calculations

## License

[Add your license here]

