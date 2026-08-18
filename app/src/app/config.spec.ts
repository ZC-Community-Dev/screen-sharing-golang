import { apiPath, eventsWsUrl, roomPath } from './config';
import { environment } from '../environments/environment';

describe('config', () => {
  it('prefixes HTTP paths with the environment apiBaseUrl', () => {
    expect(apiPath('/links')).toBe('/api/v2/links');
    expect(apiPath('links/abc')).toBe('/api/v2/links/abc');
  });

  it('builds a same-origin WebSocket URL for events', () => {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    expect(eventsWsUrl('Abcdefgh12', 'sess 1')).toBe(
      `${proto}://${location.host}/api/v2/links/Abcdefgh12/events?sessionId=sess%201`,
    );
  });

  it('exposes the unchanged public room path without participant ICE configuration', () => {
    expect(roomPath('Abcdefgh12')).toBe('/r/Abcdefgh12');
  });

  it('declares a non-empty deployment transport list containing its default', () => {
    expect(environment.allowedMediaTransports.length).toBeGreaterThan(0);
    expect(new Set(environment.allowedMediaTransports).size).toBe(
      environment.allowedMediaTransports.length,
    );
    expect(environment.allowedMediaTransports).toContain(environment.defaultMediaTransport);
  });
});
