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
import { Transaction, Wallet } from '../../graphql/types';
import { format } from 'date-fns';

interface TransactionFormData {
  walletId: string;
  type: string;
  amount: number;
  currency: string;
  description?: string;
  category?: string;
  executedAt?: string;
}

interface TransactionFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: TransactionFormData) => void;
  wallets: Wallet[];
  transaction?: Transaction | null;
}

const transactionTypes = [
  { value: 'income', label: 'Income' },
  { value: 'expense', label: 'Expense' },
  { value: 'transfer', label: 'Transfer' },
];

const categories = [
  'Food',
  'Transport',
  'Shopping',
  'Bills',
  'Entertainment',
  'Healthcare',
  'Salary',
  'Investment',
  'Other',
];

export const TransactionForm: React.FC<TransactionFormProps> = ({
  open,
  onClose,
  onSubmit,
  wallets,
  transaction,
}) => {
  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<TransactionFormData>({
    defaultValues: transaction
      ? {
          walletId: transaction.walletId,
          type: transaction.type,
          amount: transaction.amount,
          currency: transaction.currency,
          description: transaction.description || '',
          category: transaction.category || '',
          executedAt: transaction.executedAt
            ? format(new Date(transaction.executedAt), "yyyy-MM-dd'T'HH:mm")
            : '',
        }
      : {
          walletId: wallets[0]?.id || '',
          type: 'expense',
          currency: wallets[0]?.currency || 'USD',
          amount: 0,
          executedAt: format(new Date(), "yyyy-MM-dd'T'HH:mm"),
        },
  });

  useEffect(() => {
    if (transaction) {
      reset({
        walletId: transaction.walletId,
        type: transaction.type,
        amount: transaction.amount,
        currency: transaction.currency,
        description: transaction.description || '',
        category: transaction.category || '',
        executedAt: transaction.executedAt
          ? format(new Date(transaction.executedAt), "yyyy-MM-dd'T'HH:mm")
          : '',
      });
    } else {
      reset({
        walletId: wallets[0]?.id || '',
        type: 'expense',
        currency: wallets[0]?.currency || 'USD',
        amount: 0,
        executedAt: format(new Date(), "yyyy-MM-dd'T'HH:mm"),
      });
    }
  }, [transaction, wallets, reset]);

  const handleFormSubmit = (data: TransactionFormData) => {
    onSubmit(data);
    reset();
  };

  const selectedWallet = wallets.find((w) => w.id === register('walletId').value);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <form onSubmit={handleSubmit(handleFormSubmit)}>
        <DialogTitle>
          {transaction ? 'Edit Transaction' : 'Create Transaction'}
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
              {transactionTypes.map((option) => (
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
              value={selectedWallet?.currency || ''}
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
              label="Category"
              select
              fullWidth
              {...register('category')}
            >
              {categories.map((category) => (
                <MenuItem key={category} value={category}>
                  {category}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Executed At"
              type="datetime-local"
              fullWidth
              InputLabelProps={{ shrink: true }}
              {...register('executedAt')}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="contained">
            {transaction ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};






