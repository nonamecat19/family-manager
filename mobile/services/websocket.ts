import { io, Socket } from 'socket.io-client';

const WS_BASE_URL = process.env.EXPO_PUBLIC_WS_URL;

export type WebSocketEvent = {
  type: 'list_updated' | 'note_updated' | 'birthday_updated' | 'family_updated' | 'folder_updated';
  familyId: string;
  data: any;
};

class WebSocketClient {
  private socket: Socket | null = null;
  private listeners: Map<string, Set<(event: WebSocketEvent) => void>> = new Map();
  private isConnecting = false;
  private currentToken: string | null = null;
  private currentFamilyId: string | null = null;

  async connect(token: string, familyId?: string): Promise<void> {
    if (this.isConnecting || (this.socket && this.socket.connected)) {
      return;
    }

    this.isConnecting = true;
    this.currentToken = token;
    this.currentFamilyId = familyId || null;

    try {
      const socket = io(WS_BASE_URL, {
        transports: ['websocket'],
        reconnection: true,
        reconnectionAttempts: 5,
        reconnectionDelay: 1000,
      });

      socket.on('connect', () => {
        console.log('WebSocket connected');
        this.isConnecting = false;
        
        // Authenticate
        socket.emit('authenticate', {
          token,
          familyId,
        });
      });

      socket.on('authenticated', (data: { familyId?: string }) => {
        console.log('WebSocket authenticated');
        if (data.familyId && familyId) {
          socket.emit('subscribe', { familyId });
        }
      });

      socket.on('subscribed', (data: { familyId: string }) => {
        console.log('WebSocket subscribed to family:', data.familyId);
      });

      socket.on('unsubscribed', (data: { familyId: string }) => {
        console.log('WebSocket unsubscribed from family:', data.familyId);
      });

      socket.on('error', (error: { message: string }) => {
        console.error('WebSocket error:', error);
        this.isConnecting = false;
      });

      socket.on('disconnect', () => {
        console.log('WebSocket disconnected');
        this.isConnecting = false;
      });

      // Handle real-time events
      socket.on('list_updated', (data: any) => {
        this.emit('list_updated', data);
      });

      socket.on('note_updated', (data: any) => {
        this.emit('note_updated', data);
      });

      socket.on('birthday_updated', (data: any) => {
        this.emit('birthday_updated', data);
      });

      socket.on('family_updated', (data: any) => {
        this.emit('family_updated', data);
      });

      socket.on('folder_updated', (data: any) => {
        this.emit('folder_updated', data);
      });

      this.socket = socket;
    } catch (error) {
      console.error('Error connecting WebSocket:', error);
      this.isConnecting = false;
    }
  }

  subscribe(familyId: string) {
    if (this.socket && this.socket.connected) {
      this.socket.emit('subscribe', { familyId });
      this.currentFamilyId = familyId;
    }
  }

  unsubscribe(familyId: string) {
    if (this.socket && this.socket.connected) {
      this.socket.emit('unsubscribe', { familyId });
      if (this.currentFamilyId === familyId) {
        this.currentFamilyId = null;
      }
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
    if (this.socket) {
      this.socket.disconnect();
      this.socket = null;
    }
    this.listeners.clear();
    this.currentToken = null;
    this.currentFamilyId = null;
  }
}

export const wsClient = new WebSocketClient();

