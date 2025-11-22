import { Injectable } from '@nestjs/common';

export interface WebSocketEvent {
  type: 'list_updated' | 'note_updated' | 'birthday_updated' | 'family_updated' | 'folder_updated';
  familyId: string;
  data: any;
}

// Store active connections by family ID
const familyRooms = new Map<string, Set<any>>();

@Injectable()
export class WebSocketService {
  broadcastToFamily(familyId: string, event: WebSocketEvent) {
    const room = familyRooms.get(familyId);
    if (!room) return;

    const message = JSON.stringify(event);
    room.forEach((ws) => {
      if (ws.readyState === 1) {
        // WebSocket.OPEN
        ws.send(message);
      }
    });
  }

  addToFamilyRoom(familyId: string, ws: any) {
    if (!familyRooms.has(familyId)) {
      familyRooms.set(familyId, new Set());
    }
    familyRooms.get(familyId)!.add(ws);
  }

  removeFromFamilyRoom(familyId: string, ws: any) {
    const room = familyRooms.get(familyId);
    if (room) {
      room.delete(ws);
      if (room.size === 0) {
        familyRooms.delete(familyId);
      }
    }
  }
}


