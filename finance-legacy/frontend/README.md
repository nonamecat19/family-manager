# Finance Manager Frontend

A modern React frontend application for managing personal finances, built with Vite, TypeScript, Material-UI, and Apollo Client.

## Features

- **Dashboard**: Overview of finances with key metrics and recent transactions
- **Wallets**: Manage multiple wallets (cash, credit cards, crypto)
- **Transactions**: Track income, expenses, and transfers with filtering
- **Investments**: Monitor investments including crypto staking
- **Recurring Payments**: Set up and manage recurring income and expenses

## Technology Stack

- **React 18** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **Material-UI (MUI)** - Component library
- **Apollo Client** - GraphQL client
- **React Router** - Routing
- **React Hook Form** - Form handling
- **date-fns** - Date formatting

## Prerequisites

- Node.js 18+ and npm/yarn/pnpm
- Backend server running on `http://localhost:8080` (see DEVELOPMENT.md for setup)
- PostgreSQL running (via docker-compose: `docker-compose up -d postgres`)

## Installation

1. Install dependencies:
```bash
npm install
# or
yarn install
# or
pnpm install
```

## Development

Start the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
```

The app will be available at `http://localhost:5173`.

## Configuration

Create a `.env` file in the frontend directory to configure the backend URL:

```env
VITE_BACKEND_URL=http://localhost:8080
```

If not set, it defaults to `http://localhost:8080`. The GraphQL endpoint will be `${VITE_BACKEND_URL}/graphql`.

## Building for Production

Build the production bundle:

```bash
npm run build
# or
yarn build
# or
pnpm build
```

The built files will be in the `dist` directory.

Preview the production build:

```bash
npm run preview
# or
yarn preview
# or
pnpm preview
```

## Project Structure

```
frontend/
├── public/              # Static assets
├── src/
│   ├── components/      # Reusable components
│   │   ├── Layout/      # Layout components
│   │   ├── Wallet/      # Wallet components
│   │   ├── Transaction/ # Transaction components
│   │   ├── Investment/  # Investment components
│   │   └── Recurring/   # Recurring payment components
│   ├── pages/           # Page components
│   ├── graphql/         # GraphQL queries, mutations, and client
│   ├── context/         # React Context providers
│   ├── App.tsx          # Main app component
│   └── main.tsx         # Entry point
├── package.json
├── vite.config.ts
└── tsconfig.json
```

## Authentication

Currently, the app uses a placeholder authentication system. The `AuthContext` manages user state and JWT tokens stored in localStorage. For development, you may need to manually set a user ID in localStorage or update the code to use a hardcoded user ID.

To use authentication:
1. Set `userId` in localStorage: `localStorage.setItem('userId', '1')`
2. Or update the code to use a default user ID for development

## GraphQL API

The frontend connects to the GraphQL backend API. Ensure the backend is running and accessible at the configured endpoint.

### Queries
- `GET_WALLETS` - Fetch user wallets
- `GET_TRANSACTIONS` - Fetch transactions with filters
- `GET_INVESTMENTS` - Fetch user investments
- `GET_RECURRING_PAYMENTS` - Fetch recurring payments

### Mutations
- `CREATE_WALLET` - Create a new wallet
- `UPDATE_WALLET` - Update wallet details
- `DELETE_WALLET` - Delete a wallet
- `CREATE_TRANSACTION` - Create a transaction
- `CREATE_INVESTMENT` - Create an investment
- `CREATE_RECURRING` - Create a recurring payment

## Error Handling

The app includes:
- Global error boundary for catching React errors
- GraphQL error handling via Apollo Client
- User-friendly error messages
- Loading states for async operations

## Browser Support

Modern browsers that support ES2020 and React 18.

## License

[Add your license here]


