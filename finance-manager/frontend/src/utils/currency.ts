import { 
  DollarSign, 
  Euro, 
  PoundSterling, 
  Coins, 
  Banknote, 
  Wallet,
  LucideIcon 
} from 'lucide-react';

/**
 * Formats a number as currency, handling both standard ISO 4217 currency codes
 * and cryptocurrency codes (e.g., USDC, BTC, ETH).
 * 
 * @param amount - The amount to format
 * @param currency - The currency code (e.g., 'USD', 'EUR', 'USDC', 'BTC')
 * @returns Formatted currency string
 */
export const formatCurrency = (amount: number | undefined, currency: string = 'USD'): string => {
  if (amount === undefined || amount === null) {
    return 'N/A';
  }

  // List of common cryptocurrency codes that are not ISO 4217 compliant
  const cryptoCurrencies = new Set([
    'USDC', 'USDT', 'BTC', 'ETH', 'BNB', 'ADA', 'SOL', 'XRP', 'DOT', 'DOGE',
    'MATIC', 'AVAX', 'LINK', 'UNI', 'LTC', 'ATOM', 'ETC', 'XLM', 'ALGO',
    'VET', 'ICP', 'FIL', 'TRX', 'EOS', 'AAVE', 'THETA', 'CRO', 'HBAR',
    'GRT', 'FTM', 'AXS', 'SAND', 'MANA', 'ENJ', 'CHZ', 'BAT', 'ZIL', 'HOT'
  ]);

  const normalizedCurrency = currency.toUpperCase().trim();
  
  // Check if it's a cryptocurrency or try to format as standard currency
  if (cryptoCurrencies.has(normalizedCurrency)) {
    // Format cryptocurrency: use number formatting and append currency code
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 8,
    }).format(amount) + ' ' + normalizedCurrency;
  }

  // Try to format as standard ISO 4217 currency
  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: normalizedCurrency,
    }).format(amount);
  } catch (error) {
    // If currency code is invalid, fall back to number formatting with code appended
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 8,
    }).format(amount) + ' ' + normalizedCurrency;
  }
};

/**
 * Returns an appropriate icon component for a given currency code.
 * 
 * @param currencyCode - The currency code (e.g., 'USD', 'EUR', 'BTC')
 * @param currencyType - Optional currency type ('fiat' or 'crypto') to help choose icon
 * @returns Lucide icon component
 */
export const getCurrencyIcon = (currencyCode: string, currencyType?: string): LucideIcon => {
  const code = currencyCode.toUpperCase().trim();
  const type = currencyType?.toLowerCase();

  // Crypto currencies
  if (type === 'crypto') {
    // Specific crypto icons could be added here if lucide-react adds them
    return Wallet;
  }

  // Fiat currencies with specific icons
  const fiatIconMap: Record<string, LucideIcon> = {
    'USD': DollarSign,
    'EUR': Euro,
    'GBP': PoundSterling,
    'JPY': Banknote,
    'CNY': Banknote, // Chinese Yuan
  };

  if (fiatIconMap[code]) {
    return fiatIconMap[code];
  }

  // Default icons
  if (type === 'fiat') {
    return Banknote;
  }

  // Generic fallback
  return Coins;
};

