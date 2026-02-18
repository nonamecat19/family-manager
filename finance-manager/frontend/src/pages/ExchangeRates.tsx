import React, { useMemo } from 'react';
import { useQuery, useMutation } from '@apollo/client';
import {
  Container,
  Typography,
  Box,
  CircularProgress,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Card,
  CardContent,
  Button,
} from '@mui/material';
import CurrencyExchangeIcon from '@mui/icons-material/CurrencyExchange';
import RefreshIcon from '@mui/icons-material/Refresh';
import { GET_CURRENCIES } from '../graphql/queries';
import { UPDATE_EXCHANGE_RATES } from '../graphql/mutations';
import { Currency } from '../graphql/types';
import { getCurrencyIcon } from '../utils/currency';

const ExchangeRates: React.FC = () => {
  const { loading, error, data, refetch } = useQuery(GET_CURRENCIES, {
    errorPolicy: 'all',
  });

  const [updateRates, { loading: updating }] = useMutation(UPDATE_EXCHANGE_RATES, {
    onCompleted: () => {
      refetch();
    },
    onError: (err) => {
      console.error('Failed to update exchange rates:', err);
    },
  });

  const currencies: Currency[] = data?.currencies || [];

  const { fiatCurrencies, cryptoCurrencies } = useMemo(() => {
    const fiat = currencies.filter((c) => c.type === 'fiat');
    const crypto = currencies.filter((c) => c.type === 'crypto');

    // Sort by code alphabetically
    fiat.sort((a, b) => a.code.localeCompare(b.code));
    crypto.sort((a, b) => a.code.localeCompare(b.code));

    return { fiatCurrencies: fiat, cryptoCurrencies: crypto };
  }, [currencies]);

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
        <Alert severity="error">Error loading exchange rates: {error.message}</Alert>
      </Container>
    );
  }

  const renderCurrencyTable = (currencyList: Currency[], title: string) => (
    <Card sx={{ mb: 4 }}>
      <CardContent>
        <Typography variant="h5" component="h2" gutterBottom sx={{ mb: 2, fontWeight: 600 }}>
          {title}
        </Typography>
        {currencyList.length === 0 ? (
          <Alert severity="info">No {title.toLowerCase()} found.</Alert>
        ) : (
          <TableContainer component={Paper} variant="outlined">
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight="bold">
                      Code
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight="bold">
                      Name
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography variant="subtitle2" fontWeight="bold">
                      Exchange Rate
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography variant="subtitle2" fontWeight="bold">
                      USD Equivalent
                    </Typography>
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {currencyList.map((currency) => {
                  const IconComponent = getCurrencyIcon(currency.code, currency.type);
                  return (
                    <TableRow key={currency.id} hover>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <IconComponent size={18} />
                          <Typography variant="body1" fontWeight="medium">
                            {currency.code}
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>{currency.name}</TableCell>
                      <TableCell align="right">
                        <Typography variant="body1">
                          {currency.exchangeRate.toLocaleString('en-US', {
                            minimumFractionDigits: 6,
                            maximumFractionDigits: 6,
                          })}
                        </Typography>
                      </TableCell>
                      <TableCell align="right">
                        <Typography variant="body2" color="text.secondary">
                          1 {currency.code} = {currency.exchangeRate.toLocaleString('en-US', {
                            minimumFractionDigits: 6,
                            maximumFractionDigits: 6,
                          })} USD
                        </Typography>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </CardContent>
    </Card>
  );

  const handleUpdateRates = async () => {
    try {
      await updateRates();
    } catch (err) {
      // Error is handled by onError callback
    }
  };

  return (
    <Container maxWidth="lg">
      <Box display="flex" alignItems="center" justifyContent="space-between" mb={3}>
        <Box display="flex" alignItems="center">
          <CurrencyExchangeIcon sx={{ mr: 1, fontSize: 32 }} color="primary" />
          <Typography variant="h4" component="h1">
            Exchange Rates
          </Typography>
        </Box>
        <Button
          variant="contained"
          color="primary"
          startIcon={updating ? <CircularProgress size={20} color="inherit" /> : <RefreshIcon />}
          onClick={handleUpdateRates}
          disabled={updating || loading}
        >
          {updating ? 'Updating...' : 'Update Rates'}
        </Button>
      </Box>

      <Box sx={{ mb: 2 }}>
        <Alert severity="info" sx={{ mb: 3 }}>
          All exchange rates are displayed relative to USD (US Dollar). Exchange rates are stored
          in the database and can be updated by clicking the "Update Rates" button above.
        </Alert>
      </Box>

      {currencies.length === 0 ? (
        <Alert severity="warning">
          No currencies found. Please ensure currencies are configured in the system.
        </Alert>
      ) : (
        <>
          {renderCurrencyTable(fiatCurrencies, 'Fiat Currencies')}
          {renderCurrencyTable(cryptoCurrencies, 'Cryptocurrencies')}
        </>
      )}
    </Container>
  );
};

export default ExchangeRates;

