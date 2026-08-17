import { Injectable } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class PresenterTokenStore {
  private key(id: string) {
    return `presenterToken:${id}`;
  }

  save(id: string, token: string) {
    sessionStorage.setItem(this.key(id), token);
  }

  get(id: string): string | null {
    return sessionStorage.getItem(this.key(id));
  }
}
