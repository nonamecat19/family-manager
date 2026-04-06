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
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { useAuth } from '../context/AuthContext';
import { GET_RECURRING_PAYMENTS } from '../graphql/queries';
import { CREATE_RECURRING } from '../graphql/mutations';
import { RecurringCard } from '../components/Recurring/RecurringCard';
import { RecurringForm } from '../components/Recurring/RecurringForm';
import { GET_WALLETS } from '../graphql/queries';

const Recurring: React.FC = () => {
  const { userId } = useAuth();
  const [formOpen, setFormOpen] = useState(false);
  const [filterType, setFilterType] = useState<string>('all');

  const { data: walletsData } = useQuery(GET_WALLETS, {
    variables: { userId: userId || '1' },
    skip: !userId,
  });

  const wallets = walletsData?.wallets || [];

  const { loading, error, data, refetch } = useQuery(GET_RECURRING_PAYMENTS, {
    variables: { userId: userId || '1' },
    skip: !userId,
  });

  const [createRecurring] = useMutation(CREATE_RECURRING, {
    onCompleted: () => {
      setFormOpen(false);
      refetch();
    },
  });

  const handleCreate = () => {
    setFormOpen(true);
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this recurring payment?')) {
      // Note: Delete mutation not in schema, so this is a placeholder
      alert('Delete functionality requires backend mutation');
    }
  };

  const handleSubmit = async (formData: {
    walletId: string;
    type: string;
    amount: number;
    currency: string;
    description?: string;
    recurrence: string;
    startDate: string;
    endDate?: string;
  }) => {
    if (!userId) return;

    await createRecurring({
      variables: {
        userId,
        ...formData,
      },
    });
  };

  const recurringPayments = data?.recurringPayments || [];
  const filteredRecurring =
    filterType === 'all'
      ? recurringPayments
      : recurringPayments.filter((r: Recurring) => r.type === filterType);

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
        <Alert severity="error">
          Error loading recurring payments: {error.message}
        </Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="lg">
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          Recurring Payments
        </Typography>
        <Box display="flex" gap={2} alignItems="center">
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
            </Select>
          </FormControl>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreate}
            disabled={wallets.length === 0}
          >
            Create Recurring
          </Button>
        </Box>
      </Box>

      {wallets.length === 0 ? (
        <Alert severity="info">
          No wallets found. Create a wallet first to add recurring payments.
        </Alert>
      ) : filteredRecurring.length === 0 ? (
        <Alert severity="info">
          No recurring payments found. Create one to get started!
        </Alert>
      ) : (
        <Grid container spacing={3}>
          {filteredRecurring.map((recurring: Recurring) => (
            <Grid item xs={12} sm={6} md={4} key={recurring.id}>
              <RecurringCard recurring={recurring} onDelete={handleDelete} />
            </Grid>
          ))}
        </Grid>
      )}

      <RecurringForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onSubmit={handleSubmit}
        wallets={wallets}
      />
    </Container>
  );
};

export default Recurring;






