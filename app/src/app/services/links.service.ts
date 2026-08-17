import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';

export interface CreateLinkResponse {
  id: string;
  publicUrl: string;
  presenterToken: string;
}

export interface LinkPublic {
  id: string;
  state: 'waiting' | 'sharing';
  participantCount: number;
}

export interface SessionResponse {
  sessionId: string;
  role: 'presenter' | 'viewer';
  id: string;
  state: 'waiting' | 'sharing';
}

@Injectable({ providedIn: 'root' })
export class LinksService {
  private readonly http = inject(HttpClient);

  create() {
    return this.http.post<CreateLinkResponse>('/api/v1/links', {});
  }

  get(id: string) {
    return this.http.get<LinkPublic>(`/api/v1/links/${id}`);
  }

  claimPresenter(id: string, presenterToken: string) {
    return this.http.post<SessionResponse>(`/api/v1/links/${id}/presenter-sessions`, {
      presenterToken,
    });
  }

  joinViewer(id: string) {
    return this.http.post<SessionResponse>(`/api/v1/links/${id}/viewer-sessions`, {});
  }

  startShare(id: string, sessionId: string, presenterToken: string) {
    return this.http.post<LinkPublic>(`/api/v1/links/${id}/share/start`, {
      sessionId,
      presenterToken,
    });
  }

  stopShare(id: string, sessionId: string, presenterToken: string) {
    return this.http.post<LinkPublic>(`/api/v1/links/${id}/share/stop`, {
      sessionId,
      presenterToken,
    });
  }
}
