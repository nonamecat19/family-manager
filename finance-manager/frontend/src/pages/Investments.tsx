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
import { GET_INVESTMENTS } from '../graphql/queries';
import { CREATE_INVESTMENT } from '../graphql/mutations';
import { InvestmentCard } from '../components/Investment/InvestmentCard';
import { InvestmentForm } from '../components/Investment/InvestmentForm';
import { GET_WALLETS } from '../graphql/queries';
import { Investment, Wallet } from '../graphql/types';

const Investments: React.FC = () => {
  const { userId } = useAuth();
  const [formOpen, setFormOpen] = useState(false);
  const [filterStatus, setFilterStatus] = useState<string>('all');

  const { data: walletsData } = useQuery(GET_WALLETS, {
    variables: { userId: userId || '1' },
    skip: !userId,
  });

  const wallets = walletsData?.wallets || [];

  const { loading, error, data, refetch } = useQuery(GET_INVESTMENTS, {
    variables: { userId: userId || '1' },
    skip: !userId,
  });

  const [createInvestment] = useMutation(CREATE_INVESTMENT, {
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
    coin?: string;
    amount: number;
    startDate: string;
    endDate?: string;
    dailyReward?: number;
  }) => {
    if (!userId) return;

    await createInvestment({
      variables: {
        userId,
        ...formData,
      },
    });
  };

  const investments = data?.investments || [];
  const filteredInvestments =
    filterStatus === 'all'
      ? investments
      : investments.filter((i: Investment) => i.status === filterStatus);

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
        <Alert severity="error">Error loading investments: {error.message}</Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="lg">
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          Investments
        </Typography>
        <Box display="flex" gap={2} alignItems="center">
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Status</InputLabel>
            <Select
              value={filterStatus}
              label="Status"
              onChange={(e) => setFilterStatus(e.target.value)}
            >
              <MenuItem value="all">All</MenuItem>
              <MenuItem value="active">Active</MenuItem>
              <MenuItem value="completed">Completed</MenuItem>
            </Select>
          </FormControl>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreate}
            disabled={wallets.length === 0}
          >
            Create Investment
          </Button>
        </Box>
      </Box>

      {wallets.length === 0 ? (
        <Alert severity="info">
          No wallets found. Create a wallet first to add investments.
        </Alert>
      ) : filteredInvestments.length === 0 ? (
        <Alert severity="info">No investments found. Create one to get started!</Alert>
      ) : (
        <Grid container spacing={3}>
          {filteredInvestments.map((investment: Investment) => (
            <Grid item xs={12} sm={6} md={4} key={investment.id}>
              <InvestmentCard investment={investment} />
            </Grid>
          ))}
        </Grid>
      )}

      <InvestmentForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onSubmit={handleSubmit}
        wallets={wallets}
      />
    </Container>
  );
};

export default Investments;






