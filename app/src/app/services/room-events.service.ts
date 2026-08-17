import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

export interface RoomEvent {
  type: 'room.state' | 'presence' | 'signal';
  from?: string;
  payload: Record<string, unknown>;
}

@Injectable({ providedIn: 'root' })
export class RoomEventsService {
  private socket?: WebSocket;
  private pending: string[] = [];
  private openResolvers: Array<() => void> = [];

  connect(id: string, sessionId: string): Observable<RoomEvent> {
    return new Observable((subscriber) => {
      const proto = location.protocol === 'https:' ? 'wss' : 'ws';
      const socket = new WebSocket(
        `${proto}://${location.host}/api/v1/links/${id}/events?sessionId=${encodeURIComponent(sessionId)}`,
      );
      this.socket = socket;
      this.pending = [];
      socket.onopen = () => {
        this.flush(socket);
        for (const resolve of this.openResolvers.splice(0)) {
          resolve();
        }
      };
      socket.onmessage = (event) => {
        subscriber.next(JSON.parse(event.data) as RoomEvent);
      };
      socket.onerror = (err) => subscriber.error(err);
      socket.onclose = () => subscriber.complete();
      return () => {
        socket.close();
        if (this.socket === socket) {
          this.socket = undefined;
          this.pending = [];
        }
      };
    });
  }

  whenOpen(): Promise<void> {
    if (this.socket?.readyState === WebSocket.OPEN) {
      return Promise.resolve();
    }
    return new Promise((resolve) => this.openResolvers.push(resolve));
  }

  send(message: unknown) {
    const data = JSON.stringify(message);
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(data);
      return;
    }
    this.pending.push(data);
  }

  private flush(socket: WebSocket) {
    for (const frame of this.pending.splice(0)) {
      socket.send(frame);
    }
  }
}
