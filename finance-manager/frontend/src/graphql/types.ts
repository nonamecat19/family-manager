export interface User {
  id: string;
  email: string;
  createdAt: string;
}

export interface Wallet {
  id: string;
  userId: string;
  name: string;
  type: string;
  currency: string;
  balance: number;
  category?: string | null;
  createdAt: string;
}

export interface Transaction {
  id: string;
  walletId: string;
  type: string;
  amount: number;
  currency: string;
  description?: string;
  category?: string;
  createdAt: string;
  executedAt?: string;
}

export interface Recurring {
  id: string;
  userId: string;
  walletId: string;
  type: string;
  amount: number;
  currency: string;
  description?: string;
  recurrence: string;
  startDate: string;
  endDate?: string;
  nextExecution?: string;
}

export interface Investment {
  id: string;
  userId: string;
  walletId: string;
  type: string;
  coin?: string;
  amount?: number;
  startDate: string;
  endDate?: string;
  dailyReward?: number;
  status: string;
}

export interface Currency {
  id: string;
  code: string;
  type: string;
  exchangeRate: number;
  name: string;
}




