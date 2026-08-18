import { apiPath, eventsWsUrl, iceServers, roomPath } from './config';

describe('config', () => {
  it('prefixes HTTP paths with the environment apiBaseUrl', () => {
    expect(apiPath('/links')).toBe('/api/v1/links');
    expect(apiPath('links/abc')).toBe('/api/v1/links/abc');
  });

  it('builds a same-origin WebSocket URL for events', () => {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    expect(eventsWsUrl('Abcdefgh12', 'sess 1')).toBe(
      `${proto}://${location.host}/api/v1/links/Abcdefgh12/events?sessionId=sess%201`,
    );
  });

  it('exposes public room path and STUN from environment', () => {
    expect(roomPath('Abcdefgh12')).toBe('/r/Abcdefgh12');
    expect(iceServers()).toEqual([{ urls: 'stun:stun.l.google.com:19302' }]);
  });
});
