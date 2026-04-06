import React, { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { useQuery } from '@apollo/client';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  MenuItem,
  Box,
  Typography,
  CircularProgress,
} from '@mui/material';
import { Wallet, Currency } from '../../graphql/types';
import { GET_CURRENCIES } from '../../graphql/queries';
import { getCurrencyIcon } from '../../utils/currency';

interface WalletFormData {
  name: string;
  type: string;
  currency: string;
  balance?: number;
  category?: string;
}

interface WalletFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: WalletFormData) => void;
  wallet?: Wallet | null;
}

const walletTypes = [
  { value: 'cash', label: 'Cash' },
  { value: 'credit_card', label: 'Credit Card' },
  { value: 'crypto', label: 'Crypto' },
];

// Fallback currencies in case API fails
// These match the database migration - exchange rates are stored in DB and can be updated
const FALLBACK_FIAT_CURRENCIES: Currency[] = [
  { id: '1', code: 'USD', type: 'fiat', exchangeRate: 1.0, name: 'US Dollar' },
  { id: '2', code: 'UAH', type: 'fiat', exchangeRate: 0.027, name: 'Ukrainian Hryvnia' },
];

const FALLBACK_CRYPTO_CURRENCIES: Currency[] = [
  { id: '3', code: 'USDT', type: 'crypto', exchangeRate: 1.0, name: 'Tether' },
  { id: '4', code: 'USDC', type: 'crypto', exchangeRate: 1.0, name: 'USD Coin' },
  { id: '5', code: 'ETH', type: 'crypto', exchangeRate: 2500.0, name: 'Ethereum' },
  { id: '6', code: 'BTC', type: 'crypto', exchangeRate: 45000.0, name: 'Bitcoin' },
];

export const WalletForm: React.FC<WalletFormProps> = ({
  open,
  onClose,
  onSubmit,
  wallet,
}) => {
  // Fetch FIAT currencies
  const { data: fiatData, loading: fiatLoading } = useQuery(GET_CURRENCIES, {
    variables: { type: 'fiat' },
    skip: !open,
    errorPolicy: 'all', // Continue even if there's an error
  });

  // Fetch Crypto currencies
  const { data: cryptoData, loading: cryptoLoading } = useQuery(GET_CURRENCIES, {
    variables: { type: 'crypto' },
    skip: !open,
    errorPolicy: 'all', // Continue even if there's an error
  });

  // Use fallback currencies if query fails or returns empty
  const fiatCurrencies: Currency[] = 
    fiatData?.currencies && fiatData.currencies.length > 0 
      ? fiatData.currencies 
      : (!fiatLoading ? FALLBACK_FIAT_CURRENCIES : []);
  
  const cryptoCurrencies: Currency[] = 
    cryptoData?.currencies && cryptoData.currencies.length > 0 
      ? cryptoData.currencies 
      : (!cryptoLoading ? FALLBACK_CRYPTO_CURRENCIES : []);

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    watch,
    setValue,
  } = useForm<WalletFormData>({
    defaultValues: wallet
      ? {
          name: wallet.name,
          type: wallet.type,
          currency: wallet.currency,
          balance: wallet.balance,
          category: wallet.category || '',
        }
      : {
          name: '',
          type: 'cash',
          currency: 'USD',
          balance: 0,
          category: '',
        },
  });

  const watchedType = watch('type');
  const watchedCurrency = watch('currency');

  // Determine which currencies to show based on wallet type
  const isCryptoType = watchedType === 'crypto';
  const availableCurrencies = isCryptoType ? cryptoCurrencies : fiatCurrencies;
  const isLoadingCurrencies = isCryptoType ? cryptoLoading : fiatLoading;

  // Update currency when wallet type changes (only for new wallets)
  useEffect(() => {
    if (!wallet && open && availableCurrencies.length > 0) {
      const firstCurrency = availableCurrencies[0].code;
      // Only update if current currency is not in the available currencies for this type
      const currentInAvailable = availableCurrencies.some(c => c.code === watchedCurrency);
      if (!currentInAvailable) {
        setValue('currency', firstCurrency);
      }
    }
  }, [watchedType, availableCurrencies, wallet, open, watchedCurrency, setValue]);

  // Ensure currency is set correctly when editing and currencies are loaded
  useEffect(() => {
    if (wallet && open && !isLoadingCurrencies && availableCurrencies.length > 0) {
      // If the wallet's currency is not in available currencies, keep it anyway (might be from different type)
      // But ensure the form has the wallet's currency value
      if (watchedCurrency !== wallet.currency) {
        setValue('currency', wallet.currency);
      }
    }
  }, [wallet, open, isLoadingCurrencies, availableCurrencies, watchedCurrency, setValue]);

  // Reset form when wallet changes or dialog opens
  useEffect(() => {
    if (open) {
      if (wallet) {
        // When editing, set all fields including type and currency
        reset({
          name: wallet.name || '',
          type: wallet.type || 'cash',
          currency: wallet.currency || 'USD',
          balance: wallet.balance || 0,
          category: wallet.category || '',
        });
      } else {
        // When creating, set defaults
        reset({
          name: '',
          type: 'cash',
          currency: 'USD',
          balance: 0,
          category: '',
        });
      }
    }
  }, [wallet, reset, open]);

  const handleFormSubmit = async (data: WalletFormData) => {
    console.log('WalletForm submitting with data:', data, 'wallet:', wallet);
    await onSubmit(data);
    reset();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <form onSubmit={handleSubmit(handleFormSubmit)}>
        <DialogTitle>{wallet ? 'Edit Wallet' : 'Create Wallet'}</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
            <TextField
              label="Name"
              fullWidth
              {...register('name', { required: 'Name is required' })}
              error={!!errors.name}
              helperText={errors.name?.message}
            />
            <TextField
              label="Type"
              select
              fullWidth
              value={watchedType || ''}
              {...register('type', { required: 'Type is required' })}
              error={!!errors.type}
              helperText={errors.type?.message}
            >
              {walletTypes.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </TextField>
            
            {/* Currency Selection - automatically filtered by wallet type */}
            <Box>
              <Typography variant="body2" sx={{ mb: 1, color: 'text.secondary' }}>
                Currency {isCryptoType ? '(Crypto)' : '(FIAT)'}
              </Typography>
              
              {isLoadingCurrencies ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                  <CircularProgress size={24} />
                </Box>
              ) : availableCurrencies.length > 0 ? (
                <TextField
                  select
                  fullWidth
                  value={watchedCurrency || ''}
                  {...register('currency', { required: 'Currency is required' })}
                  error={!!errors.currency}
                  helperText={errors.currency?.message}
                >
                  {availableCurrencies.map((currency) => {
                    const IconComponent = getCurrencyIcon(currency.code, currency.type);
                    return (
                      <MenuItem key={currency.code} value={currency.code}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <IconComponent size={18} />
                          <Box>
                            <Typography variant="body1">
                              {currency.code} - {currency.name}
                            </Typography>
                            {currency.code !== 'USD' && currency.exchangeRate !== 1 && (
                              <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                                1 {currency.code} = {currency.exchangeRate.toLocaleString('en-US', {
                                  minimumFractionDigits: 2,
                                  maximumFractionDigits: 2,
                                })} USD
                              </Typography>
                            )}
                          </Box>
                        </Box>
                      </MenuItem>
                    );
                  })}
                </TextField>
              ) : (
                <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>
                  Loading currencies...
                </Typography>
              )}
            </Box>

            <TextField
              label="Balance"
              type="number"
              fullWidth
              inputProps={{ step: '0.0000000000001' }}
              {...register('balance', {
                valueAsNumber: true,
              })}
              error={!!errors.balance}
              helperText={errors.balance?.message || 'Enter balance (supports very small decimal values)'}
            />
            <TextField
              label="Category (optional)"
              fullWidth
              {...register('category')}
              error={!!errors.category}
              helperText={errors.category?.message || 'Group wallets by category'}
              placeholder="e.g., Personal, Business, Savings"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="contained">
            {wallet ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};
