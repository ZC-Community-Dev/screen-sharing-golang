import { apiPath, applyMediaUdpHost, applyMediaUdpMtu, eventsWsUrl, isPrivateIPv4, mediaIceConfiguration, mediaUdpMtu, roomPath, webrtcUdpMayBeUnreachable } from './config';
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
    expect(mediaIceConfiguration()).toEqual({ iceServers: [], iceCandidatePoolSize: 0 });
  });

  it('declares a non-empty deployment transport list containing its default', () => {
    expect(environment.allowedMediaTransports.length).toBeGreaterThan(0);
    expect(new Set(environment.allowedMediaTransports).size).toBe(
      environment.allowedMediaTransports.length,
    );
    expect(environment.allowedMediaTransports).toContain(environment.defaultMediaTransport);
  });

  it('keeps server SDP unchanged when mediaUdpHost is empty', () => {
    const sdp = [
      'c=IN IP4 10.0.0.8',
      'a=candidate:1 1 udp 2130706431 10.0.0.8 5000 typ host',
      '',
    ].join('\r\n');
    expect(applyMediaUdpHost(sdp, '', 5000)).toBe(sdp);
    expect(environment.mediaUdpPort).toBeGreaterThanOrEqual(1);
    expect(environment.mediaUdpPort).toBeLessThanOrEqual(65535);
  });

  it('rewrites host ICE candidates to the configured remote UDP endpoint', () => {
    const sdp = [
      'c=IN IP4 127.0.0.1',
      'a=candidate:1 1 udp 2130706431 127.0.0.1 9 typ host generation 0',
      'a=ice-lite',
      '',
    ].join('\n');
    expect(applyMediaUdpHost(sdp, '192.168.10.108', 5000)).toBe(
      [
        'c=IN IP4 192.168.10.108',
        'a=candidate:1 1 udp 2130706431 192.168.10.108 5000 typ host generation 0',
        'a=ice-lite',
        '',
      ].join('\n'),
    );
  });

  it('rejects an invalid remote UDP host or port', () => {
    expect(() => applyMediaUdpHost('c=IN IP4 127.0.0.1', 'not-an-ip', 5000)).toThrow(
      /IPv4/,
    );
    expect(() => applyMediaUdpHost('c=IN IP4 127.0.0.1', '192.168.10.108', 0)).toThrow(
      /65535/,
    );
  });

  it('keeps WebRTC UDP packets at the configured 1200-byte MTU', () => {
    expect(environment.mediaUdpMtu).toBe(1200);
    expect(mediaUdpMtu()).toBe(1200);
    expect(() => mediaUdpMtu(1500)).toThrow(/1200/);
    const sdp = ['m=video 9 UDP/TLS/RTP/SAVPF 96', 'a=fmtp:96 max-fs=3600', ''].join('\n');
    const limited = applyMediaUdpMtu(sdp, 1200);
    expect(limited).toContain('x-google-max-bitrate=2500');
    expect(limited).toContain('a=x-google-flag:conference');
  });

  it('treats RFC1918 and loopback hosts as private IPv4', () => {
    expect(isPrivateIPv4('192.168.17.138')).toBe(true);
    expect(isPrivateIPv4('10.0.0.8')).toBe(true);
    expect(isPrivateIPv4('127.0.0.1')).toBe(true);
    expect(isPrivateIPv4('8.8.8.8')).toBe(false);
  });

  it('flags WebRTC/UDP as unreachable from a public hostname to a private ICE host', () => {
    expect(webrtcUdpMayBeUnreachable('192.168.17.138', 'share.zappycraft.com')).toBe(true);
    expect(webrtcUdpMayBeUnreachable('192.168.17.138', '192.168.17.138')).toBe(false);
    expect(webrtcUdpMayBeUnreachable('', 'share.zappycraft.com')).toBe(false);
  });
});
