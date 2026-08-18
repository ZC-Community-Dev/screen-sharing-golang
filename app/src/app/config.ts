import { environment } from '../environments/environment';

export function apiPath(path: string): string {
  const base = environment.apiBaseUrl.replace(/\/$/, '');
  const suffix = path.startsWith('/') ? path : `/${path}`;
  return `${base}${suffix}`;
}

export function eventsWsUrl(linkId: string, sessionId: string): string {
  const path = apiPath(
    `/links/${encodeURIComponent(linkId)}/events?sessionId=${encodeURIComponent(sessionId)}`,
  );
  if (path.startsWith('https://')) {
    return `wss://${path.slice('https://'.length)}`;
  }
  if (path.startsWith('http://')) {
    return `ws://${path.slice('http://'.length)}`;
  }
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  return `${proto}://${location.host}${path}`;
}

export function publicOrigin(): string {
  return environment.appOrigin || location.origin;
}

export function roomPath(id: string): string {
  const prefix = environment.roomPathPrefix.replace(/\/$/, '');
  return `${prefix}/${id}`;
}

export function iceServers(): RTCIceServer[] {
  return environment.stunUrls.map((urls) => ({ urls }));
}
