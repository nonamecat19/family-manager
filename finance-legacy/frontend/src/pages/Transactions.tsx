import React, { useState } from 'react';
import { useQuery, useMutation } from '@apollo/client';
import {
  Container,
  Typography,
  Button,
  Box,
  CircularProgress,
  Alert,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  TextField,
  MenuItem,
  FormControl,
  InputLabel,
  Select,
  IconButton,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { useAuth } from '../context/AuthContext';
import { GET_WALLETS } from '../graphql/queries';
import { GET_TRANSACTIONS } from '../graphql/queries';
import { CREATE_TRANSACTION } from '../graphql/mutations';
import { TransactionForm } from '../components/Transaction/TransactionForm';
import { Transaction, Wallet } from '../graphql/types';
import { format } from 'date-fns';
import { formatCurrency } from '../utils/currency';

const Transactions: React.FC = () => {
  const { userId } = useAuth();
  const [formOpen, setFormOpen] = useState(false);
  const [selectedWalletId, setSelectedWalletId] = useState<string>('');
  const [filterType, setFilterType] = useState<string>('all');
  const [dateFrom, setDateFrom] = useState<string>('');
  const [dateTo, setDateTo] = useState<string>('');

  const { data: walletsData } = useQuery(GET_WALLETS, {
    variables: { userId: userId || '1' },
    skip: !userId,
  });

  const wallets = walletsData?.wallets || [];

  const { loading, error, data, refetch } = useQuery(GET_TRANSACTIONS, {
    variables: {
      walletId: selectedWalletId || wallets[0]?.id || '',
      from: dateFrom || undefined,
      to: dateTo || undefined,
    },
    skip: !selectedWalletId && wallets.length === 0,
  });

  const [createTransaction] = useMutation(CREATE_TRANSACTION, {
    onCompleted: () => {
      setFormOpen(false);
      refetch();
    },
  });

  const handleCreate = () => {
    setFormOpen(true);
  };

  const handleSubmit = async (formData: {
    walletId: string;
    type: string;
    amount: number;
    currency: string;
    description?: string;
    category?: string;
    executedAt?: string;
  }) => {
    await createTransaction({
      variables: {
        ...formData,
        executedAt: formData.executedAt || undefined,
      },
    });
  };

  const transactions = data?.transactions || [];
  const filteredTransactions =
    filterType === 'all'
      ? transactions
      : transactions.filter((t: Transaction) => t.type === filterType);

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

  React.useEffect(() => {
    if (wallets.length > 0 && !selectedWalletId) {
      setSelectedWalletId(wallets[0].id);
    }
  }, [wallets, selectedWalletId]);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Container>
        <Alert severity="error">Error loading transactions: {error.message}</Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="lg">
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          Transactions
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleCreate}
          disabled={wallets.length === 0}
        >
          Create Transaction
        </Button>
      </Box>

      <Box display="flex" gap={2} mb={3} flexWrap="wrap">
        <FormControl size="small" sx={{ minWidth: 200 }}>
          <InputLabel>Wallet</InputLabel>
          <Select
            value={selectedWalletId}
            label="Wallet"
            onChange={(e) => setSelectedWalletId(e.target.value)}
          >
            {wallets.map((wallet: Wallet) => (
              <MenuItem key={wallet.id} value={wallet.id}>
                {wallet.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <FormControl size="small" sx={{ minWidth: 120 }}>
          <InputLabel>Type</InputLabel>
          <Select
            value={filterType}
            label="Type"
            onChange={(e) => setFilterType(e.target.value)}
          >
            <MenuItem value="all">All</MenuItem>
            <MenuItem value="income">Income</MenuItem>
            <MenuItem value="expense">Expense</MenuItem>
            <MenuItem value="transfer">Transfer</MenuItem>
          </Select>
        </FormControl>
        <TextField
          label="From Date"
          type="date"
          size="small"
          value={dateFrom}
          onChange={(e) => setDateFrom(e.target.value)}
          InputLabelProps={{ shrink: true }}
        />
        <TextField
          label="To Date"
          type="date"
          size="small"
          value={dateTo}
          onChange={(e) => setDateTo(e.target.value)}
          InputLabelProps={{ shrink: true }}
        />
      </Box>

      {wallets.length === 0 ? (
        <Alert severity="info">
          No wallets found. Create a wallet first to add transactions.
        </Alert>
      ) : filteredTransactions.length === 0 ? (
        <Alert severity="info">No transactions found.</Alert>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Date</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Amount</TableCell>
                <TableCell>Description</TableCell>
                <TableCell>Category</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredTransactions.map((transaction: Transaction) => (
                <TableRow key={transaction.id}>
                  <TableCell>
                    {transaction.executedAt
                      ? format(new Date(transaction.executedAt), 'MMM dd, yyyy HH:mm')
                      : format(new Date(transaction.createdAt), 'MMM dd, yyyy HH:mm')}
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
                  <TableCell>{transaction.description || '-'}</TableCell>
                  <TableCell>{transaction.category || '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <TransactionForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onSubmit={handleSubmit}
        wallets={wallets}
      />
    </Container>
  );
};

export default Transactions;




