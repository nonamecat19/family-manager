import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { apiClient } from '@/services/api';
import { wsClient } from '@/services/websocket';
import { useAuth } from './AuthContext';

interface Family {
  id: string;
  name: string;
  icon?: string;
  color?: string;
  role: 'owner' | 'member';
  createdAt: string;
}

interface FamilyContextType {
  families: Family[];
  activeFamily: Family | null;
  loading: boolean;
  setActiveFamily: (family: Family) => Promise<void>;
  refreshFamilies: () => Promise<void>;
}

const FamilyContext = createContext<FamilyContextType | undefined>(undefined);

export function FamilyProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  const [families, setFamilies] = useState<Family[]>([]);
  const [activeFamily, setActiveFamilyState] = useState<Family | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (isAuthenticated) {
      loadFamilies();
    }
  }, [isAuthenticated]);

  async function loadFamilies() {
    try {
      const data = await apiClient.getFamilies();
      setFamilies(data);
      
      // Load active family from storage
      const storedFamilyId = await AsyncStorage.getItem('active_family_id');
      if (storedFamilyId) {
        const family = data.find((f) => f.id === storedFamilyId);
        if (family) {
          setActiveFamilyState(family);
          // Subscribe to family room
          wsClient.subscribe(family.id);
        }
      } else if (data.length > 0) {
        // Set first family as active
        await setActiveFamily(data[0]);
      }
    } catch (error) {
      console.error('Failed to load families:', error);
    } finally {
      setLoading(false);
    }
  }

  async function setActiveFamily(family: Family) {
    // Unsubscribe from previous family
    if (activeFamily) {
      wsClient.unsubscribe(activeFamily.id);
    }

    setActiveFamilyState(family);
    await AsyncStorage.setItem('active_family_id', family.id);
    
    // Subscribe to new family room
    wsClient.subscribe(family.id);
    
    // Update session on server
    try {
      await apiClient.switchFamily(family.id);
    } catch (error) {
      console.error('Failed to switch family on server:', error);
    }
  }

  async function refreshFamilies() {
    await loadFamilies();
  }

  return (
    <FamilyContext.Provider
      value={{
        families,
        activeFamily,
        loading,
        setActiveFamily,
        refreshFamilies,
      }}
    >
      {children}
    </FamilyContext.Provider>
  );
}

export function useFamily() {
  const context = useContext(FamilyContext);
  if (context === undefined) {
    throw new Error('useFamily must be used within a FamilyProvider');
  }
  return context;
}

