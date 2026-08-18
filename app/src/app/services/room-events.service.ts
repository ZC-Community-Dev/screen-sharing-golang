import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { eventsWsUrl } from '../config';
import { PublicationPublic, PublicationState, RoomState } from './links.service';
import { MediaTransport } from './media-transport';

export type RoomEvent =
  | { type: 'room.state'; payload: { state: RoomState; publication?: PublicationPublic | null } }
  | {
      type: 'publication.state';
      payload: {
        publicationId: string;
        transport: MediaTransport;
        state: PublicationState;
      };
    }
  | { type: 'presence'; payload: { participantCount: number } }
  | {
      type: 'media.state';
      payload: {
        state: 'connecting' | 'connected' | 'reconnecting' | 'failed' | 'closed';
        role: 'presenter' | 'viewer';
        mediaSessionId?: string;
        transport?: MediaTransport;
        publicationId?: string;
      };
    };

@Injectable({ providedIn: 'root' })
export class RoomEventsService {
  connect(id: string, sessionId: string): Observable<RoomEvent> {
    return new Observable((subscriber) => {
      const socket = new WebSocket(eventsWsUrl(id, sessionId));
      socket.onmessage = (event) => {
        subscriber.next(JSON.parse(event.data) as RoomEvent);
      };
      socket.onerror = (err) => subscriber.error(err);
      socket.onclose = () => subscriber.complete();
      return () => {
        socket.close();
      };
    });
  }
}
