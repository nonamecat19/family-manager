import React from 'react';
import { useQuery } from '@apollo/client';
import {
  Container,
  Typography,
  Box,
  Grid,
  Card,
  CardContent,
  CircularProgress,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
} from '@mui/material';
import AccountBalanceWalletIcon from '@mui/icons-material/AccountBalanceWallet';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import ReceiptIcon from '@mui/icons-material/Receipt';
import RepeatIcon from '@mui/icons-material/Repeat';
import { useAuth } from '../context/AuthContext';
import { GET_WALLETS } from '../graphql/queries';
import { GET_TRANSACTIONS } from '../graphql/queries';
import { GET_INVESTMENTS } from '../graphql/queries';
import { GET_RECURRING_PAYMENTS } from '../graphql/queries';
import { GET_CURRENCIES } from '../graphql/queries';
import { Wallet, Transaction, Investment, Currency } from '../graphql/types';
import { format } from 'date-fns';
import { formatCurrency } from '../utils/currency';

const Dashboard: React.FC = () => {
  const { userId } = useAuth();

  const { loading: walletsLoading, data: walletsData } = useQuery(GET_WALLETS, {
    variables: { userId: userId || '1' },
    skip: !userId,
  });

  const { loading: transactionsLoading, data: transactionsData } = useQuery(
    GET_TRANSACTIONS,
    {
      variables: {
        walletId: walletsData?.wallets?.[0]?.id || '',
        from: undefined,
        to: undefined,
      },
      skip: !walletsData?.wallets?.[0]?.id,
    }
  );

  const { loading: investmentsLoading, data: investmentsData } = useQuery(
    GET_INVESTMENTS,
    {
      variables: { userId: userId || '1' },
      skip: !userId,
    }
  );

  const { loading: recurringLoading, data: recurringData } = useQuery(
    GET_RECURRING_PAYMENTS,
    {
      variables: { userId: userId || '1' },
      skip: !userId,
    }
  );

  const { loading: currenciesLoading, data: currenciesData } = useQuery(GET_CURRENCIES, {
    errorPolicy: 'all',
  });

  const loading =
    walletsLoading || transactionsLoading || investmentsLoading || recurringLoading || currenciesLoading;

  const wallets = walletsData?.wallets || [];
  const transactions = transactionsData?.transactions || [];
  const investments = investmentsData?.investments || [];
  const recurringPayments = recurringData?.recurringPayments || [];
  const currencies = currenciesData?.currencies || [];

  // Create a map of currency code to exchange rate for quick lookup
  const currencyMap = new Map<string, number>();
  currencies.forEach((currency: Currency) => {
    currencyMap.set(currency.code.toUpperCase(), currency.exchangeRate);
  });

  // Calculate total balance in USD by converting each wallet's balance
  const totalBalanceUSD = wallets.reduce((sum: number, wallet: Wallet) => {
    const exchangeRate = currencyMap.get(wallet.currency.toUpperCase()) ?? 1.0;
    return sum + wallet.balance * exchangeRate;
  }, 0);

  const totalBalance = totalBalanceUSD;

  // Calculate totals grouped by currency
  const totalsByCurrency = wallets.reduce((acc: Record<string, number>, wallet: Wallet) => {
    const currency = wallet.currency.toUpperCase();
    if (!acc[currency]) {
      acc[currency] = 0;
    }
    acc[currency] += wallet.balance;
    return acc;
  }, {});

  const activeInvestments = investments.filter(
    (inv: Investment) => inv.status === 'active'
  );

  const recentTransactions = transactions
    .slice()
    .sort((a: Transaction, b: Transaction) => {
      const dateA = new Date(a.executedAt || a.createdAt).getTime();
      const dateB = new Date(b.executedAt || b.createdAt).getTime();
      return dateB - dateA;
    })
    .slice(0, 5);

  const totalIncome = transactions
    .filter((t: Transaction) => t.type === 'income')
    .reduce((sum: number, t: Transaction) => sum + t.amount, 0);

  const totalExpenses = transactions
    .filter((t: Transaction) => t.type === 'expense')
    .reduce((sum: number, t: Transaction) => sum + t.amount, 0);

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'income':
        return 'success';
      case 'expense':
        return 'error';
      case 'transfer':
        return 'info';
      default:
        return 'default';
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Container maxWidth="lg">
      <Typography variant="h4" component="h1" gutterBottom>
        Dashboard
      </Typography>

      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" mb={1}>
                <AccountBalanceWalletIcon color="primary" sx={{ mr: 1 }} />
                <Typography variant="h6">Total Balance</Typography>
              </Box>
              <Typography variant="h4" color="primary">
                {formatCurrency(totalBalance, 'USD')}
              </Typography>
              {Object.keys(totalsByCurrency).length > 0 && (
                <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                  {Object.entries(totalsByCurrency)
                    .map(([currency, amount]) => `${formatCurrency(amount, currency)}`)
                    .join(', ')}
                </Typography>
              )}
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                {wallets.length} wallet{wallets.length !== 1 ? 's' : ''}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" mb={1}>
                <TrendingUpIcon color="success" sx={{ mr: 1 }} />
                <Typography variant="h6">Active Investments</Typography>
              </Box>
              <Typography variant="h4" color="success.main">
                {activeInvestments.length}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {investments.length} total
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" mb={1}>
                <ReceiptIcon color="success" sx={{ mr: 1 }} />
                <Typography variant="h6">Total Income</Typography>
              </Box>
              <Typography variant="h4" color="success.main">
                {formatCurrency(totalIncome, 'USD')}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {transactions.filter((t: Transaction) => t.type === 'income').length}{' '}
                transactions
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" mb={1}>
                <ReceiptIcon color="error" sx={{ mr: 1 }} />
                <Typography variant="h6">Total Expenses</Typography>
              </Box>
              <Typography variant="h4" color="error.main">
                {formatCurrency(totalExpenses, 'USD')}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {transactions.filter((t: Transaction) => t.type === 'expense').length}{' '}
                transactions
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Grid container spacing={3}>
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Recent Transactions
              </Typography>
              {recentTransactions.length === 0 ? (
                <Alert severity="info">No recent transactions</Alert>
              ) : (
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Date</TableCell>
                        <TableCell>Type</TableCell>
                        <TableCell>Amount</TableCell>
                        <TableCell>Description</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {recentTransactions.map((transaction: Transaction) => (
                        <TableRow key={transaction.id}>
                          <TableCell>
                            {transaction.executedAt
                              ? format(
                                  new Date(transaction.executedAt),
                                  'MMM dd, yyyy'
                                )
                              : format(new Date(transaction.createdAt), 'MMM dd, yyyy')}
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={transaction.type}
                              color={getTypeColor(transaction.type) as any}
                              size="small"
                            />
                          </TableCell>
                          <TableCell>
                            <Typography
                              color={
                                transaction.type === 'income'
                                  ? 'success.main'
                                  : transaction.type === 'expense'
                                  ? 'error.main'
                                  : 'text.primary'
                              }
                            >
                              {transaction.type === 'expense' ? '-' : '+'}
                              {formatCurrency(transaction.amount, transaction.currency)}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            {transaction.description || '-'}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card sx={{ mb: 2 }}>
            <CardContent>
              <Box display="flex" alignItems="center" mb={1}>
                <RepeatIcon color="primary" sx={{ mr: 1 }} />
                <Typography variant="h6">Recurring Payments</Typography>
              </Box>
              <Typography variant="h3" color="primary">
                {recurringPayments.length}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Active recurring payments
              </Typography>
            </CardContent>
          </Card>

          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Wallet Summary
              </Typography>
              {wallets.length === 0 ? (
                <Alert severity="info">No wallets</Alert>
              ) : (
                <Box>
                  {wallets.slice(0, 3).map((wallet: Wallet) => (
                    <Box
                      key={wallet.id}
                      display="flex"
                      justifyContent="space-between"
                      mb={1}
                    >
                      <Typography variant="body2">{wallet.name}</Typography>
                      <Typography variant="body2" fontWeight="bold">
                        {formatCurrency(wallet.balance, wallet.currency)}
                      </Typography>
                    </Box>
                  ))}
                  {wallets.length > 3 && (
                    <Typography variant="caption" color="text.secondary">
                      +{wallets.length - 3} more
                    </Typography>
                  )}
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Container>
  );
};

export default Dashboard;




