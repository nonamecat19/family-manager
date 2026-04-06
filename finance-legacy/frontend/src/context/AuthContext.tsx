import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface AuthContextType {
  userId: string | null;
  token: string | null;
  setAuth: (userId: string, token: string) => void;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [userId, setUserId] = useState<string | null>(() => {
    // For development: use stored userId or default to '1'
    return localStorage.getItem('userId') || '1';
  });
  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem('token');
  });

  useEffect(() => {
    if (userId) {
      localStorage.setItem('userId', userId);
    }
    if (token) {
      localStorage.setItem('token', token);
    }
  }, [userId, token]);

  const setAuth = (newUserId: string, newToken: string) => {
    setUserId(newUserId);
    setToken(newToken);
    localStorage.setItem('userId', newUserId);
    localStorage.setItem('token', newToken);
  };

  const logout = () => {
    setUserId(null);
    setToken(null);
    localStorage.removeItem('userId');
    localStorage.removeItem('token');
  };

  const value: AuthContextType = {
    userId,
    token,
    setAuth,
    logout,
    isAuthenticated: !!userId && !!token,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

