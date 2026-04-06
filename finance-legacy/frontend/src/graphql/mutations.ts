import { gql } from '@apollo/client';

export const CREATE_WALLET = gql`
  mutation CreateWallet($userId: ID!, $name: String!, $type: String!, $currency: String!, $balance: Float, $category: String) {
    createWallet(userId: $userId, name: $name, type: $type, currency: $currency, balance: $balance, category: $category) {
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

export const UPDATE_WALLET = gql`
  mutation UpdateWallet($id: ID!, $name: String, $type: String, $currency: String, $balance: Float, $category: String) {
    updateWallet(id: $id, name: $name, type: $type, currency: $currency, balance: $balance, category: $category) {
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

export const DELETE_WALLET = gql`
  mutation DeleteWallet($id: ID!) {
    deleteWallet(id: $id)
  }
`;

export const CREATE_TRANSACTION = gql`
  mutation CreateTransaction(
    $walletId: ID!
    $type: String!
    $amount: Float!
    $currency: String!
    $description: String
    $category: String
    $executedAt: String
  ) {
    createTransaction(
      walletId: $walletId
      type: $type
      amount: $amount
      currency: $currency
      description: $description
      category: $category
      executedAt: $executedAt
    ) {
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

export const CREATE_INVESTMENT = gql`
  mutation CreateInvestment(
    $userId: ID!
    $walletId: ID!
    $type: String!
    $coin: String
    $amount: Float!
    $startDate: String!
    $endDate: String
    $dailyReward: Float
  ) {
    createInvestment(
      userId: $userId
      walletId: $walletId
      type: $type
      coin: $coin
      amount: $amount
      startDate: $startDate
      endDate: $endDate
      dailyReward: $dailyReward
    ) {
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

export const CREATE_RECURRING = gql`
  mutation CreateRecurring(
    $userId: ID!
    $walletId: ID!
    $type: String!
    $amount: Float!
    $currency: String!
    $description: String
    $recurrence: String!
    $startDate: String!
    $endDate: String
  ) {
    createRecurring(
      userId: $userId
      walletId: $walletId
      type: $type
      amount: $amount
      currency: $currency
      description: $description
      recurrence: $recurrence
      startDate: $startDate
      endDate: $endDate
    ) {
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

export const UPDATE_EXCHANGE_RATES = gql`
  mutation UpdateExchangeRates {
    updateExchangeRates
  }
`;




