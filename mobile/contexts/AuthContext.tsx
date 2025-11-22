import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { apiClient } from '@/services/api';
import { wsClient } from '@/services/websocket';

interface User {
  id: string;
  email: string;
  name?: string;
  avatar?: string;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name?: string) => Promise<void>;
  logout: () => Promise<void>;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    checkAuth();
  }, []);

  async function checkAuth() {
    try {
      const token = await AsyncStorage.getItem('auth_token');
      if (token) {
        const userData = await apiClient.get<{ id: string; email: string; name?: string; avatar?: string }>('/auth/me');
        setUser(userData);
        // Connect WebSocket
        wsClient.connect(token);
      }
    } catch (error) {
      console.error('Auth check failed:', error);
      await AsyncStorage.removeItem('auth_token');
    } finally {
      setLoading(false);
    }
  }

  async function login(email: string, password: string) {
    const response = await apiClient.login(email, password);
    setUser(response.user);
    // Connect WebSocket
    wsClient.connect(response.token);
  }

  async function register(email: string, password: string, name?: string) {
    const response = await apiClient.register(email, password, name);
    setUser(response.user);
    // Connect WebSocket
    wsClient.connect(response.token);
  }

  async function logout() {
    await apiClient.logout();
    setUser(null);
    wsClient.disconnect();
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        login,
        register,
        logout,
        isAuthenticated: !!user,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

