import React from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Chip,
  IconButton,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import { Recurring } from '../../graphql/types';
import { format } from 'date-fns';
import { formatCurrency } from '../../utils/currency';

interface RecurringCardProps {
  recurring: Recurring;
  onDelete: (id: string) => void;
}

const getTypeColor = (type: string) => {
  return type === 'income' ? 'success' : 'error';
};

const getRecurrenceLabel = (recurrence: string) => {
  switch (recurrence) {
    case 'daily':
      return 'Daily';
    case 'weekly':
      return 'Weekly';
    case 'monthly':
      return 'Monthly';
    default:
      return recurrence;
  }
};

export const RecurringCard: React.FC<RecurringCardProps> = ({
  recurring,
  onDelete,
}) => {
  return (
    <Card sx={{ minWidth: 275, mb: 2 }}>
      <CardContent>
        <Box display="flex" justifyContent="space-between" alignItems="flex-start">
          <Box flex={1}>
            <Typography variant="h6" component="div" gutterBottom>
              {recurring.description || 'Recurring Payment'}
            </Typography>
            <Typography variant="h5" color="primary" gutterBottom>
              {formatCurrency(recurring.amount, recurring.currency)}
            </Typography>
            <Box mt={1} display="flex" gap={1} flexWrap="wrap">
              <Chip
                label={recurring.type}
                color={getTypeColor(recurring.type) as any}
                size="small"
              />
              <Chip
                label={getRecurrenceLabel(recurring.recurrence)}
                size="small"
                variant="outlined"
              />
              <Chip label={recurring.currency} size="small" variant="outlined" />
            </Box>
            <Box mt={2}>
              <Typography variant="body2" color="text.secondary">
                Start: {format(new Date(recurring.startDate), 'MMM dd, yyyy')}
              </Typography>
              {recurring.endDate && (
                <Typography variant="body2" color="text.secondary">
                  End: {format(new Date(recurring.endDate), 'MMM dd, yyyy')}
                </Typography>
              )}
              {recurring.nextExecution && (
                <Typography variant="body2" color="primary" sx={{ mt: 0.5 }}>
                  Next: {format(new Date(recurring.nextExecution), 'MMM dd, yyyy')}
                </Typography>
              )}
            </Box>
          </Box>
          <IconButton
            size="small"
            onClick={() => onDelete(recurring.id)}
            color="error"
          >
            <DeleteIcon />
          </IconButton>
        </Box>
      </CardContent>
    </Card>
  );
};




