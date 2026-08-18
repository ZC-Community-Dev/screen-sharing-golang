import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';

import { apiPath } from '../config';

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
    return this.http.post<CreateLinkResponse>(apiPath('/links'), {});
  }

  get(id: string) {
    return this.http.get<LinkPublic>(apiPath(`/links/${id}`));
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
}
