import React from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Chip,
  IconButton,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { Wallet } from '../../graphql/types';
import { formatCurrency, getCurrencyIcon } from '../../utils/currency';

interface WalletCardProps {
  wallet: Wallet;
  onEdit: (wallet: Wallet) => void;
  onDelete: (id: string) => void;
  currencyMap?: Map<string, number>;
}

const getWalletTypeColor = (type: string) => {
  switch (type) {
    case 'cash':
      return 'success';
    case 'credit_card':
      return 'warning';
    case 'crypto':
      return 'info';
    default:
      return 'default';
  }
};

export const WalletCard: React.FC<WalletCardProps> = ({
  wallet,
  onEdit,
  onDelete,
  currencyMap,
}) => {
  const exchangeRate = currencyMap?.get(wallet.currency.toUpperCase()) ?? 1.0;
  const usdEquivalent = wallet.balance * exchangeRate;

  return (
    <Card sx={{ minWidth: 275, mb: 2 }}>
      <CardContent>
        <Box display="flex" justifyContent="space-between" alignItems="flex-start">
          <Box>
            <Typography variant="h6" component="div" gutterBottom>
              {wallet.name}
            </Typography>
            <Typography variant="h4" color="primary" gutterBottom>
              {formatCurrency(wallet.balance, wallet.currency)}
            </Typography>
            {wallet.currency.toUpperCase() !== 'USD' && (
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                ≈ {formatCurrency(usdEquivalent, 'USD')}
              </Typography>
            )}
            <Box mt={1} display="flex" gap={1} flexWrap="wrap">
              <Chip
                label={wallet.type}
                color={getWalletTypeColor(wallet.type) as any}
                size="small"
              />
              <Chip 
                label={wallet.currency} 
                size="small" 
                variant="outlined"
                icon={
                  (() => {
                    const IconComponent = getCurrencyIcon(
                      wallet.currency, 
                      wallet.type === 'crypto' ? 'crypto' : 'fiat'
                    );
                    return <IconComponent size={14} />;
                  })()
                }
              />
              {wallet.category && (
                <Chip 
                  label={wallet.category} 
                  size="small" 
                  color="secondary" 
                  variant="outlined"
                />
              )}
            </Box>
          </Box>
          <Box>
            <IconButton
              size="small"
              onClick={() => onEdit(wallet)}
              color="primary"
            >
              <EditIcon />
            </IconButton>
            <IconButton
              size="small"
              onClick={() => onDelete(wallet.id)}
              color="error"
            >
              <DeleteIcon />
            </IconButton>
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
};




