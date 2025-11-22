import { apiClient } from './api';

const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:3000';

export type WebSocketEvent = {
  type: 'list_updated' | 'note_updated' | 'birthday_updated' | 'family_updated' | 'folder_updated';
  familyId: string;
  data: any;
};

type WebSocketMessage = 
  | { type: 'authenticate'; token: string; familyId?: string }
  | { type: 'subscribe'; familyId: string }
  | { type: 'unsubscribe'; familyId: string };

class WebSocketClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private listeners: Map<string, Set<(event: WebSocketEvent) => void>> = new Map();
  private isConnecting = false;

  async connect(token: string, familyId?: string): Promise<void> {
    if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
      return;
    }

    this.isConnecting = true;

    try {
      const ws = new WebSocket(`${WS_BASE_URL}/ws`);
      
      ws.onopen = () => {
        console.log('WebSocket connected');
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        
        // Authenticate
        ws.send(JSON.stringify({
          type: 'authenticate',
          token,
          familyId,
        }));

        if (familyId) {
          // Subscribe to family room
          ws.send(JSON.stringify({
            type: 'subscribe',
            familyId,
          }));
        }
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          
          if (data.type === 'authenticated' || data.type === 'subscribed' || data.type === 'unsubscribed') {
            console.log('WebSocket:', data.type);
            return;
          }

          // Handle real-time events
          if (data.type && data.familyId) {
            this.emit(data.type, data);
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.isConnecting = false;
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        this.isConnecting = false;
        this.ws = null;
        this.reconnect(token, familyId);
      };

      this.ws = ws;
    } catch (error) {
      console.error('Error connecting WebSocket:', error);
      this.isConnecting = false;
      this.reconnect(token, familyId);
    }
  }

  private reconnect(token: string, familyId?: string) {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      console.log(`Reconnecting... (attempt ${this.reconnectAttempts})`);
      this.connect(token, familyId);
    }, delay);
  }

  subscribe(familyId: string) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'subscribe',
        familyId,
      }));
    }
  }

  unsubscribe(familyId: string) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'unsubscribe',
        familyId,
      }));
    }
  }

  on(eventType: string, callback: (event: WebSocketEvent) => void) {
    if (!this.listeners.has(eventType)) {
      this.listeners.set(eventType, new Set());
    }
    this.listeners.get(eventType)!.add(callback);
  }

  off(eventType: string, callback: (event: WebSocketEvent) => void) {
    const callbacks = this.listeners.get(eventType);
    if (callbacks) {
      callbacks.delete(callback);
    }
  }

  private emit(eventType: string, event: WebSocketEvent) {
    const callbacks = this.listeners.get(eventType);
    if (callbacks) {
      callbacks.forEach((callback) => callback(event));
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.listeners.clear();
    this.reconnectAttempts = 0;
  }
}

export const wsClient = new WebSocketClient();

