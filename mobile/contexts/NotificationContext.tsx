import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { wsClient } from '@/services/websocket';
import { useAuth } from './AuthContext';
import { useFamily } from './FamilyContext';

export interface Notification {
  id: string;
  type: 'invitation_accepted' | 'member_added' | 'family_updated' | 'general';
  title: string;
  message: string;
  timestamp: Date;
  read: boolean;
  data?: any;
}

interface NotificationContextType {
  notifications: Notification[];
  unreadCount: number;
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) => void;
  markAsRead: (id: string) => void;
  markAllAsRead: () => void;
  clearNotification: (id: string) => void;
  clearAll: () => void;
}

const NotificationContext = createContext<NotificationContextType | undefined>(undefined);

export function NotificationProvider({ children }: { children: ReactNode }) {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const { isAuthenticated } = useAuth();
  const { activeFamily, refreshFamilies } = useFamily();

  function addNotification(notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) {
    const newNotification: Notification = {
      ...notification,
      id: `${Date.now()}-${Math.random()}`,
      timestamp: new Date(),
      read: false,
    };

    setNotifications((prev) => [newNotification, ...prev]);
  }

  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    // Listen for family_updated events which include invitation acceptance
    const handleFamilyUpdate = (event: any) => {
      if (event.type === 'family_updated' && event.data) {
        const { action, userId, family } = event.data;

        if (action === 'member_added') {
          // Someone joined the family (accepted invitation)
          addNotification({
            type: 'invitation_accepted',
            title: 'New Member Joined',
            message: `A new member has joined ${activeFamily?.name || 'the workspace'}`,
            data: { userId, familyId: event.familyId },
          });
          
          // Refresh families list
          refreshFamilies();
        } else if (action === 'member_updated') {
          addNotification({
            type: 'family_updated',
            title: 'Member Role Updated',
            message: 'A member\'s role has been updated',
            data: { userId, familyId: event.familyId },
          });
        } else if (action === 'member_removed') {
          addNotification({
            type: 'family_updated',
            title: 'Member Removed',
            message: 'A member has been removed from the workspace',
            data: { userId, familyId: event.familyId },
          });
          refreshFamilies();
        }
      }
    };

    wsClient.on('family_updated', handleFamilyUpdate);

    return () => {
      wsClient.off('family_updated', handleFamilyUpdate);
    };
  }, [isAuthenticated, activeFamily, refreshFamilies]);

  function markAsRead(id: string) {
    setNotifications((prev) =>
      prev.map((n) => (n.id === id ? { ...n, read: true } : n))
    );
  }

  function markAllAsRead() {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
  }

  function clearNotification(id: string) {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  }

  function clearAll() {
    setNotifications([]);
  }

  const unreadCount = notifications.filter((n) => !n.read).length;

  return (
    <NotificationContext.Provider
      value={{
        notifications,
        unreadCount,
        addNotification,
        markAsRead,
        markAllAsRead,
        clearNotification,
        clearAll,
      }}
    >
      {children}
    </NotificationContext.Provider>
  );
}

export function useNotifications() {
  const context = useContext(NotificationContext);
  if (context === undefined) {
    throw new Error('useNotifications must be used within a NotificationProvider');
  }
  return context;
}

