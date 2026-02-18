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
import { Recurring, Wallet } from '../../graphql/types';
import { format } from 'date-fns';

interface RecurringFormData {
  walletId: string;
  type: string;
  amount: number;
  currency: string;
  description?: string;
  recurrence: string;
  startDate: string;
  endDate?: string;
}

interface RecurringFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: RecurringFormData) => void;
  wallets: Wallet[];
  recurring?: Recurring | null;
}

const recurringTypes = [
  { value: 'income', label: 'Income' },
  { value: 'expense', label: 'Expense' },
];

const recurrences = [
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
];

export const RecurringForm: React.FC<RecurringFormProps> = ({
  open,
  onClose,
  onSubmit,
  wallets,
  recurring,
}) => {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
    reset,
  } = useForm<RecurringFormData>({
    defaultValues: recurring
      ? {
          walletId: recurring.walletId,
          type: recurring.type,
          amount: recurring.amount,
          currency: recurring.currency,
          description: recurring.description || '',
          recurrence: recurring.recurrence,
          startDate: format(new Date(recurring.startDate), 'yyyy-MM-dd'),
          endDate: recurring.endDate
            ? format(new Date(recurring.endDate), 'yyyy-MM-dd')
            : '',
        }
      : {
          walletId: wallets[0]?.id || '',
          type: 'expense',
          currency: wallets[0]?.currency || 'USD',
          amount: 0,
          recurrence: 'monthly',
          startDate: format(new Date(), 'yyyy-MM-dd'),
          endDate: '',
        },
  });

  const selectedWallet = watch('walletId');
  const wallet = wallets.find((w) => w.id === selectedWallet);

  useEffect(() => {
    if (recurring) {
      reset({
        walletId: recurring.walletId,
        type: recurring.type,
        amount: recurring.amount,
        currency: recurring.currency,
        description: recurring.description || '',
        recurrence: recurring.recurrence,
        startDate: format(new Date(recurring.startDate), 'yyyy-MM-dd'),
        endDate: recurring.endDate
          ? format(new Date(recurring.endDate), 'yyyy-MM-dd')
          : '',
      });
    } else {
      reset({
        walletId: wallets[0]?.id || '',
        type: 'expense',
        currency: wallets[0]?.currency || 'USD',
        amount: 0,
        recurrence: 'monthly',
        startDate: format(new Date(), 'yyyy-MM-dd'),
        endDate: '',
      });
    }
  }, [recurring, wallets, reset]);

  useEffect(() => {
    if (wallet) {
      setValue('currency', wallet.currency);
    }
  }, [selectedWallet, wallet, setValue]);

  const handleFormSubmit = (data: RecurringFormData) => {
    onSubmit({
      ...data,
      endDate: data.endDate || undefined,
    });
    reset();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <form onSubmit={handleSubmit(handleFormSubmit)}>
        <DialogTitle>
          {recurring ? 'Edit Recurring Payment' : 'Create Recurring Payment'}
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
              {recurringTypes.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </TextField>
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
              label="Currency"
              fullWidth
              value={wallet?.currency || ''}
              disabled
            />
            <TextField
              label="Description"
              fullWidth
              multiline
              rows={2}
              {...register('description')}
            />
            <TextField
              label="Recurrence"
              select
              fullWidth
              {...register('recurrence', { required: 'Recurrence is required' })}
              error={!!errors.recurrence}
              helperText={errors.recurrence?.message}
            >
              {recurrences.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </TextField>
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
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="contained">
            {recurring ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

