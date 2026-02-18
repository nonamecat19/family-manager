import React from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Chip,
  LinearProgress,
} from '@mui/material';
import { Investment } from '../../graphql/types';
import { format } from 'date-fns';
import { formatCurrency } from '../../utils/currency';

interface InvestmentCardProps {
  investment: Investment;
}

export const InvestmentCard: React.FC<InvestmentCardProps> = ({ investment }) => {
  const isActive = investment.status === 'active';
  const startDate = new Date(investment.startDate);
  const endDate = investment.endDate ? new Date(investment.endDate) : null;
  const now = new Date();
  const progress = endDate
    ? Math.min(
        100,
        Math.max(
          0,
          ((now.getTime() - startDate.getTime()) /
            (endDate.getTime() - startDate.getTime())) *
            100
        )
      )
    : 0;

  return (
    <Card sx={{ minWidth: 275, mb: 2 }}>
      <CardContent>
        <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={2}>
          <Box>
            <Typography variant="h6" component="div" gutterBottom>
              {investment.type === 'crypto_stake' ? 'Crypto Staking' : investment.type}
            </Typography>
            {investment.coin && (
              <Chip label={investment.coin.toUpperCase()} size="small" sx={{ mb: 1 }} />
            )}
          </Box>
          <Chip
            label={investment.status}
            color={isActive ? 'success' : 'default'}
            size="small"
          />
        </Box>

        <Box mb={2}>
          <Typography variant="body2" color="text.secondary">
            Amount
          </Typography>
          <Typography variant="h5" color="primary">
            {formatCurrency(investment.amount)}
          </Typography>
        </Box>

        {investment.dailyReward && (
          <Box mb={2}>
            <Typography variant="body2" color="text.secondary">
              Daily Reward
            </Typography>
            <Typography variant="h6">
              {formatCurrency(investment.dailyReward)}
            </Typography>
          </Box>
        )}

        <Box mb={2}>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            Start Date: {format(startDate, 'MMM dd, yyyy')}
          </Typography>
          {endDate && (
            <Typography variant="body2" color="text.secondary">
              End Date: {format(endDate, 'MMM dd, yyyy')}
            </Typography>
          )}
        </Box>

        {isActive && endDate && (
          <Box>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              Progress
            </Typography>
            <LinearProgress variant="determinate" value={progress} />
            <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5 }}>
              {Math.round(progress)}% complete
            </Typography>
          </Box>
        )}
      </CardContent>
    </Card>
  );
};




