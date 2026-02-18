import { gql } from '@apollo/client';

export const GET_WALLETS = gql`
  query GetWallets($userId: ID!) {
    wallets(userId: $userId) {
      id
      userId
      name
      type
      currency
      balance
      category
      createdAt
    }
  }
`;

export const GET_TRANSACTIONS = gql`
  query GetTransactions($walletId: ID!, $from: String, $to: String) {
    transactions(walletId: $walletId, from: $from, to: $to) {
      id
      walletId
      type
      amount
      currency
      description
      category
      createdAt
      executedAt
    }
  }
`;

export const GET_INVESTMENTS = gql`
  query GetInvestments($userId: ID!) {
    investments(userId: $userId) {
      id
      userId
      walletId
      type
      coin
      amount
      startDate
      endDate
      dailyReward
      status
    }
  }
`;

export const GET_RECURRING_PAYMENTS = gql`
  query GetRecurringPayments($userId: ID!) {
    recurringPayments(userId: $userId) {
      id
      userId
      walletId
      type
      amount
      currency
      description
      recurrence
      startDate
      endDate
      nextExecution
    }
  }
`;

export const GET_CURRENCIES = gql`
  query GetCurrencies($type: String) {
    currencies(type: $type) {
      id
      code
      type
      exchangeRate
      name
    }
  }
`;




