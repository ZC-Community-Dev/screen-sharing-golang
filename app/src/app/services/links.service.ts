import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';

import { apiPath } from '../config';
import { MediaTransport, PublicMediaConfig } from './media-transport';

export type RoomState = 'waiting' | 'connecting' | 'sharing' | 'reconnecting' | 'failed';
export type PublicationState = 'connecting' | 'live' | 'reconnecting' | 'failed' | 'ended';

export interface PublicationPublic {
  id: string;
  transport: MediaTransport;
  state: PublicationState;
}

export interface CreateLinkResponse {
  id: string;
  publicUrl: string;
  presenterToken: string;
}

export interface LinkPublic {
  id: string;
  state: RoomState;
  participantCount: number;
  publication?: PublicationPublic | null;
}

export interface SessionResponse {
  sessionId: string;
  role: 'presenter' | 'viewer';
  id: string;
  state: RoomState;
}

export interface SessionDescription {
  type: 'offer' | 'answer';
  sdp: string;
}

export interface PublisherOffer {
  sessionId: string;
  presenterToken: string;
  offer: SessionDescription;
}

export interface SubscriberOffer {
  sessionId: string;
  offer: SessionDescription;
}

export interface MediaAnswer {
  mediaSessionId: string;
  answer: SessionDescription;
}

export interface WebSocketTicketRequest {
  sessionId: string;
  role: 'publisher' | 'viewer';
  presenterToken?: string;
}

export interface WebSocketTicket {
  ticket: string;
  expiresAt: string;
  websocketPath: string;
}

@Injectable({ providedIn: 'root' })
export class LinksService {
  private readonly http = inject(HttpClient);

  create() {
    return this.http.post<CreateLinkResponse>(apiPath('/links'), {});
  }

  get(id: string) {
    return this.http.get<LinkPublic>(apiPath(`/links/${id}`));
  }

  getMediaConfig() {
    return this.http.get<PublicMediaConfig>(apiPath('/media/config'));
  }

  claimPresenter(id: string, presenterToken: string) {
    return this.http.post<SessionResponse>(apiPath(`/links/${id}/presenter-sessions`), {
      presenterToken,
    });
  }

  joinViewer(id: string) {
    return this.http.post<SessionResponse>(apiPath(`/links/${id}/viewer-sessions`), {});
  }

  startShare(id: string, sessionId: string, presenterToken: string) {
    return this.http.post<LinkPublic>(apiPath(`/links/${id}/share/start`), {
      sessionId,
      presenterToken,
    });
  }

  stopShare(id: string, sessionId: string, presenterToken: string) {
    return this.http.post<LinkPublic>(apiPath(`/links/${id}/share/stop`), {
      sessionId,
      presenterToken,
    });
  }

  createPublisher(id: string, request: PublisherOffer) {
    return this.http.post<MediaAnswer>(apiPath(`/links/${id}/media/publisher`), request);
  }

  createSubscriber(id: string, request: SubscriberOffer) {
    return this.http.post<MediaAnswer>(apiPath(`/links/${id}/media/subscribers`), request);
  }

  deleteSubscriber(id: string, mediaSessionId: string, sessionId: string) {
    return this.http.delete<void>(
      apiPath(`/links/${id}/media/subscribers/${encodeURIComponent(mediaSessionId)}`),
      { headers: { 'X-Room-Session-ID': sessionId } },
    );
  }

  createWebSocketTicket(id: string, request: WebSocketTicketRequest) {
    return this.http.post<WebSocketTicket>(
      apiPath(`/links/${id}/media/websocket-tickets`),
      request,
    );
  }
}
