import {
  WebSocketGateway,
  WebSocketServer,
  SubscribeMessage,
  OnGatewayConnection,
  OnGatewayDisconnect,
  ConnectedSocket,
  MessageBody,
} from '@nestjs/websockets';
import { Server, Socket } from 'socket.io';
import { JwtService } from '@nestjs/jwt';
import { ConfigService } from '@nestjs/config';
import { DatabaseService } from '../database/database.service';
import { users } from '../database/schema';
import { eq } from 'drizzle-orm';
import { WebSocketService } from './websocket.service';

interface AuthenticatedSocket extends Socket {
  userId?: string;
  familyId?: string;
}

@WebSocketGateway({
  cors: {
    origin: process.env.CORS_ORIGIN?.split(',') || [
      'http://localhost:8081',
      'http://localhost:1420',
      'http://localhost:5173',
      'http://127.0.0.1:1420',
      'http://127.0.0.1:5173',
    ],
    credentials: true,
  },
})
export class FamilyManagerWebSocketGateway implements OnGatewayConnection, OnGatewayDisconnect {
  @WebSocketServer()
  server: Server;

  constructor(
    private jwtService: JwtService,
    private configService: ConfigService,
    private databaseService: DatabaseService,
    private websocketService: WebSocketService,
  ) {}

  async handleConnection(client: AuthenticatedSocket) {
    console.log('WebSocket connection opened:', client.id);
  }

  async handleDisconnect(client: AuthenticatedSocket) {
    console.log('WebSocket connection closed:', client.id);
    if (client.familyId) {
      this.websocketService.removeFromFamilyRoom(client.familyId, client);
    }
  }

  @SubscribeMessage('authenticate')
  async handleAuthenticate(
    @ConnectedSocket() client: AuthenticatedSocket,
    @MessageBody() data: { token: string; familyId?: string },
  ) {
    try {
      const secret = this.configService.get<string>('JWT_SECRET') || 'your-secret-key';
      const payload = await this.jwtService.verifyAsync(data.token, { secret });

      if (!payload || !payload.userId) {
        client.emit('error', { message: 'Invalid token' });
        client.disconnect();
        return;
      }

      const user = await this.databaseService.db.query.users.findFirst({
        where: eq(users.id, payload.userId),
      });

      if (!user) {
        client.emit('error', { message: 'User not found' });
        client.disconnect();
        return;
      }

      client.userId = user.id;

      if (data.familyId) {
        client.familyId = data.familyId;
        this.websocketService.addToFamilyRoom(data.familyId, client);
      }

      client.emit('authenticated', { familyId: data.familyId });
    } catch (error) {
      client.emit('error', { message: 'Authentication failed' });
      client.disconnect();
    }
  }

  @SubscribeMessage('subscribe')
  handleSubscribe(
    @ConnectedSocket() client: AuthenticatedSocket,
    @MessageBody() data: { familyId: string },
  ) {
    if (!client.userId) {
      client.emit('error', { message: 'Not authenticated' });
      return;
    }

    // Remove from previous family room if exists
    if (client.familyId && client.familyId !== data.familyId) {
      this.websocketService.removeFromFamilyRoom(client.familyId, client);
    }

    // Add to new family room
    client.familyId = data.familyId;
    this.websocketService.addToFamilyRoom(data.familyId, client);
    client.emit('subscribed', { familyId: data.familyId });
  }

  @SubscribeMessage('unsubscribe')
  handleUnsubscribe(
    @ConnectedSocket() client: AuthenticatedSocket,
    @MessageBody() data: { familyId: string },
  ) {
    if (client.familyId === data.familyId) {
      this.websocketService.removeFromFamilyRoom(data.familyId, client);
      client.familyId = undefined;
    }
    client.emit('unsubscribed', { familyId: data.familyId });
  }
}

