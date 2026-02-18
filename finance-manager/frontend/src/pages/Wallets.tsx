import React, { useState } from 'react';
import { useQuery, useMutation } from '@apollo/client';
import {
  Container,
  Typography,
  Button,
  Box,
  Grid,
  CircularProgress,
  Alert,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { useAuth } from '../context/AuthContext';
import { GET_WALLETS, GET_CURRENCIES } from '../graphql/queries';
import { CREATE_WALLET, UPDATE_WALLET, DELETE_WALLET } from '../graphql/mutations';
import { WalletCard } from '../components/Wallet/WalletCard';
import { WalletForm } from '../components/Wallet/WalletForm';
import { Wallet, Currency } from '../graphql/types';
import { formatCurrency } from '../utils/currency';

const Wallets: React.FC = () => {
  const { userId } = useAuth();
  const [formOpen, setFormOpen] = useState(false);
  const [editingWallet, setEditingWallet] = useState<Wallet | null>(null);
  const [filterType, setFilterType] = useState<string>('all');

  const { loading, error, data, refetch } = useQuery(GET_WALLETS, {
    variables: { userId: userId || '1' },
    skip: !userId,
    fetchPolicy: 'cache-and-network',
    notifyOnNetworkStatusChange: true,
  });

  const { loading: currenciesLoading, data: currenciesData } = useQuery(GET_CURRENCIES, {
    errorPolicy: 'all',
  });

  const [createWallet] = useMutation(CREATE_WALLET, {
    refetchQueries: [
      {
        query: GET_WALLETS,
        variables: { userId: userId || '1' },
      },
    ],
    onCompleted: () => {
      setFormOpen(false);
    },
    onError: (error) => {
      console.error('Error creating wallet:', error);
      alert(`Error creating wallet: ${error.message}`);
    },
  });

  const [updateWallet] = useMutation(UPDATE_WALLET, {
    refetchQueries: [
      {
        query: GET_WALLETS,
        variables: { userId: userId || '1' },
      },
    ],
    awaitRefetchQueries: true,
    onCompleted: async (data) => {
      console.log('Update mutation completed, data:', data);
      // Explicit refetch as backup
      try {
        await refetch();
        console.log('Explicit refetch completed');
      } catch (err) {
        console.error('Error in explicit refetch:', err);
      }
      setFormOpen(false);
      setEditingWallet(null);
    },
    onError: (error) => {
      console.error('Error updating wallet:', error);
      alert(`Error updating wallet: ${error.message}`);
    },
  });

  const [deleteWallet] = useMutation(DELETE_WALLET, {
    refetchQueries: [
      {
        query: GET_WALLETS,
        variables: { userId: userId || '1' },
      },
    ],
  });

  const handleCreate = () => {
    setEditingWallet(null);
    setFormOpen(true);
  };

  const handleEdit = (wallet: Wallet) => {
    setEditingWallet(wallet);
    setFormOpen(true);
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this wallet?')) {
      await deleteWallet({ variables: { id } });
    }
  };

  const handleSubmit = async (formData: {
    name: string;
    type: string;
    currency: string;
    balance?: number;
    category?: string;
  }) => {
    if (!userId) return;

    try {
      if (editingWallet) {
        console.log('Updating wallet with data:', { id: editingWallet.id, ...formData });
        await updateWallet({
          variables: {
            id: editingWallet.id,
            name: formData.name,
            type: formData.type,
            currency: formData.currency,
            balance: formData.balance,
            category: formData.category || null,
          },
        });
      } else {
        await createWallet({
          variables: {
            userId,
            ...formData,
            category: formData.category || null,
          },
        });
      }
    } catch (error) {
      console.error('Error in handleSubmit:', error);
    }
  };

  const wallets = data?.wallets || [];
  const currencies = currenciesData?.currencies || [];
  
  // Create a map of currency code to exchange rate for quick lookup
  const currencyMap = new Map<string, number>();
  currencies.forEach((currency: Currency) => {
    currencyMap.set(currency.code.toUpperCase(), currency.exchangeRate);
  });

  const filteredWallets =
    filterType === 'all'
      ? wallets
      : wallets.filter((w: Wallet) => w.type === filterType);

  // Group wallets by category
  const groupedWallets = filteredWallets.reduce((acc: Record<string, Wallet[]>, wallet: Wallet) => {
    const category = wallet.category || 'Uncategorized';
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(wallet);
    return acc;
  }, {});

  // Calculate total in USD for each category
  const categoryTotals = Object.keys(groupedWallets).reduce((acc: Record<string, number>, category: string) => {
    const categoryWallets = groupedWallets[category];
    const total = categoryWallets.reduce((sum: number, wallet: Wallet) => {
      const exchangeRate = currencyMap.get(wallet.currency.toUpperCase()) ?? 1.0;
      return sum + wallet.balance * exchangeRate;
    }, 0);
    acc[category] = total;
    return acc;
  }, {});

  const categoryKeys = Object.keys(groupedWallets).sort((a, b) => {
    // Put "Uncategorized" at the end
    if (a === 'Uncategorized') return 1;
    if (b === 'Uncategorized') return -1;
    return a.localeCompare(b);
  });

  if (loading || currenciesLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Container>
        <Alert severity="error">Error loading wallets: {error.message}</Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="lg">
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          Wallets
        </Typography>
        <Box display="flex" gap={2} alignItems="center">
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Filter</InputLabel>
            <Select
              value={filterType}
              label="Filter"
              onChange={(e) => setFilterType(e.target.value)}
            >
              <MenuItem value="all">All</MenuItem>
              <MenuItem value="cash">Cash</MenuItem>
              <MenuItem value="credit_card">Credit Card</MenuItem>
              <MenuItem value="crypto">Crypto</MenuItem>
            </Select>
          </FormControl>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreate}
          >
            Create Wallet
          </Button>
        </Box>
      </Box>

      {filteredWallets.length === 0 ? (
        <Alert severity="info">No wallets found. Create one to get started!</Alert>
      ) : (
        <Box>
          {categoryKeys.map((category) => (
            <Box key={category} sx={{ mb: 4 }}>
              <Box display="flex" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
                <Typography variant="h5" component="h2" sx={{ fontWeight: 600 }}>
                  {category}
                </Typography>
                <Typography variant="body1" color="text.secondary">
                  Total: {formatCurrency(categoryTotals[category] || 0, 'USD')}
                </Typography>
              </Box>
              <Grid container spacing={3}>
                {groupedWallets[category].map((wallet: Wallet) => (
                  <Grid item xs={12} sm={6} md={4} key={`${wallet.id}-${wallet.name}-${wallet.balance}-${wallet.currency}`}>
                    <WalletCard
                      wallet={wallet}
                      onEdit={handleEdit}
                      onDelete={handleDelete}
                      currencyMap={currencyMap}
                    />
                  </Grid>
                ))}
              </Grid>
            </Box>
          ))}
        </Box>
      )}

      <WalletForm
        open={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditingWallet(null);
        }}
        onSubmit={handleSubmit}
        wallet={editingWallet}
      />
    </Container>
  );
};

export default Wallets;


