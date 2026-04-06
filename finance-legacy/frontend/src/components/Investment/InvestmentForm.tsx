import React, { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  MenuItem,
  Box,
} from '@mui/material';
import { Investment, Wallet } from '../../graphql/types';
import { format } from 'date-fns';

interface InvestmentFormData {
  walletId: string;
  type: string;
  coin?: string;
  amount: number;
  startDate: string;
  endDate?: string;
  dailyReward?: number;
}

interface InvestmentFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: InvestmentFormData) => void;
  wallets: Wallet[];
  investment?: Investment | null;
}

const investmentTypes = [
  { value: 'crypto_stake', label: 'Crypto Staking' },
  { value: 'other', label: 'Other' },
];

const coins = ['BTC', 'ETH', 'USDT', 'BNB', 'ADA', 'SOL', 'DOT', 'MATIC'];

export const InvestmentForm: React.FC<InvestmentFormProps> = ({
  open,
  onClose,
  onSubmit,
  wallets,
  investment,
}) => {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
    reset,
  } = useForm<InvestmentFormData>({
    defaultValues: investment
      ? {
          walletId: investment.walletId,
          type: investment.type,
          coin: investment.coin || '',
          amount: investment.amount || 0,
          startDate: format(new Date(investment.startDate), 'yyyy-MM-dd'),
          endDate: investment.endDate
            ? format(new Date(investment.endDate), 'yyyy-MM-dd')
            : '',
          dailyReward: investment.dailyReward || 0,
        }
      : {
          walletId: wallets[0]?.id || '',
          type: 'crypto_stake',
          coin: '',
          amount: 0,
          startDate: format(new Date(), 'yyyy-MM-dd'),
          endDate: '',
          dailyReward: 0,
        },
  });

  const investmentType = watch('type');

  useEffect(() => {
    if (investment) {
      reset({
        walletId: investment.walletId,
        type: investment.type,
        coin: investment.coin || '',
        amount: investment.amount || 0,
        startDate: format(new Date(investment.startDate), 'yyyy-MM-dd'),
        endDate: investment.endDate
          ? format(new Date(investment.endDate), 'yyyy-MM-dd')
          : '',
        dailyReward: investment.dailyReward || 0,
      });
    } else {
      reset({
        walletId: wallets[0]?.id || '',
        type: 'crypto_stake',
        coin: '',
        amount: 0,
        startDate: format(new Date(), 'yyyy-MM-dd'),
        endDate: '',
        dailyReward: 0,
      });
    }
  }, [investment, wallets, reset]);

  const handleFormSubmit = (data: InvestmentFormData) => {
    onSubmit({
      ...data,
      endDate: data.endDate || undefined,
      dailyReward: data.dailyReward || undefined,
      coin: data.coin || undefined,
    });
    reset();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <form onSubmit={handleSubmit(handleFormSubmit)}>
        <DialogTitle>
          {investment ? 'Edit Investment' : 'Create Investment'}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
            <TextField
              label="Wallet"
              select
              fullWidth
              {...register('walletId', { required: 'Wallet is required' })}
              error={!!errors.walletId}
              helperText={errors.walletId?.message}
            >
              {wallets.map((wallet) => (
                <MenuItem key={wallet.id} value={wallet.id}>
                  {wallet.name} ({wallet.currency})
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Type"
              select
              fullWidth
              {...register('type', { required: 'Type is required' })}
              error={!!errors.type}
              helperText={errors.type?.message}
            >
              {investmentTypes.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </TextField>
            {investmentType === 'crypto_stake' && (
              <TextField
                label="Coin"
                select
                fullWidth
                {...register('coin')}
              >
                {coins.map((coin) => (
                  <MenuItem key={coin} value={coin}>
                    {coin}
                  </MenuItem>
                ))}
              </TextField>
            )}
            <TextField
              label="Amount"
              type="number"
              fullWidth
              inputProps={{ step: '0.01', min: '0' }}
              {...register('amount', {
                required: 'Amount is required',
                valueAsNumber: true,
                min: { value: 0.01, message: 'Amount must be greater than 0' },
              })}
              error={!!errors.amount}
              helperText={errors.amount?.message}
            />
            <TextField
              label="Start Date"
              type="date"
              fullWidth
              InputLabelProps={{ shrink: true }}
              {...register('startDate', { required: 'Start date is required' })}
              error={!!errors.startDate}
              helperText={errors.startDate?.message}
            />
            <TextField
              label="End Date (optional)"
              type="date"
              fullWidth
              InputLabelProps={{ shrink: true }}
              {...register('endDate')}
            />
            <TextField
              label="Daily Reward (optional)"
              type="number"
              fullWidth
              inputProps={{ step: '0.01', min: '0' }}
              {...register('dailyReward', {
                valueAsNumber: true,
              })}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="contained">
            {investment ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};






