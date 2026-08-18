import { environment } from '../environments/environment';

const IPV4 =
  /^(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)$/;

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

export function mediaWsUrl(websocketPath: string, ticket: string): string {
  const path = `${websocketPath}?ticket=${encodeURIComponent(ticket)}`;
  if (/^https?:\/\//.test(path)) {
    return path.replace(/^http/, 'ws');
  }
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  return `${proto}://${location.host}${path.startsWith('/') ? path : `/${path}`}`;
}

export function publicOrigin(): string {
  return environment.appOrigin || location.origin;
}

export function roomPath(id: string): string {
  const prefix = environment.roomPathPrefix.replace(/\/$/, '');
  return `${prefix}/${id}`;
}

/** ICE Lite SFU: browsers connect only to the configured host, never P2P. */
export function mediaIceConfiguration(): RTCConfiguration {
  return { iceServers: [], iceCandidatePoolSize: 0 };
}

export function mediaUdpMtu(mtu = environment.mediaUdpMtu): number {
  if (!Number.isInteger(mtu) || mtu < 576 || mtu > 1200) {
    throw new Error('mediaUdpMtu deve estar entre 576 e 1200.');
  }
  return mtu;
}

/** Screen-share encodings that keep RTP/SRTP datagrams near `mediaUdpMtu`. */
export function mediaUdpSendEncodings(mtu = environment.mediaUdpMtu): RTCRtpEncodingParameters[] {
  mediaUdpMtu(mtu);
  return [
    {
      maxBitrate: 2_500_000,
      maxFramerate: 15,
      priority: 'high',
      networkPriority: 'high',
      scaleResolutionDownBy: 1,
    },
  ];
}

/**
 * Asks Chromium to start conservatively and packetize for a 1200-byte path MTU.
 */
export function applyMediaUdpMtu(sdp: string, mtu = environment.mediaUdpMtu): string {
  mediaUdpMtu(mtu);
  const extras =
    'x-google-max-bitrate=2500;x-google-start-bitrate=400;x-google-min-bitrate=100';
  let next = sdp.replace(/^a=fmtp:(\d+) (.+)$/gm, (line, pt: string, rest: string) => {
    if (rest.includes('x-google-max-bitrate')) return line;
    return `a=fmtp:${pt} ${rest};${extras}`;
  });
  if (!/^a=x-google-flag:conference\s*$/m.test(next)) {
    next = next.replace(/^(m=video[^\r\n]*)/m, `$1\r\na=x-google-flag:conference`);
  }
  return next;
}

/**
 * Rewrites host ICE candidates and the media connection address so the
 * browser sends RTP to `mediaUdpHost:mediaUdpPort`. Empty host leaves SDP as-is.
 */
export function applyMediaUdpHost(
  sdp: string,
  host = environment.mediaUdpHost,
  port = environment.mediaUdpPort,
): string {
  const endpoint = host.trim();
  if (!endpoint) return sdp;
  if (!IPV4.test(endpoint)) {
    throw new Error('mediaUdpHost deve ser um IPv4 alcançável para UDP.');
  }
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error('mediaUdpPort deve estar entre 1 e 65535.');
  }
  return sdp
    .replace(/^c=IN IP4 \S+/gm, `c=IN IP4 ${endpoint}`)
    .replace(
      /^a=candidate:(\S+) (\d+) (\S+) (\d+) \S+ \d+ typ host(.*)$/gm,
      (
        _match,
        foundation: string,
        component: string,
        transport: string,
        priority: string,
        rest: string,
      ) =>
        `a=candidate:${foundation} ${component} ${transport} ${priority} ${endpoint} ${port} typ host${rest}`,
    );
}

export function isPrivateIPv4(host: string): boolean {
  const endpoint = host.trim();
  if (!IPV4.test(endpoint)) return false;
  const octets = endpoint.split('.').map(Number);
  const a = octets[0] ?? 0;
  const b = octets[1] ?? 0;
  return a === 10 || a === 127 || (a === 192 && b === 168) || (a === 172 && b >= 16 && b <= 31);
}

/** True when ICE would target a private host from a public page (Cloudflare/HTTPS). */
export function webrtcUdpMayBeUnreachable(
  host = environment.mediaUdpHost,
  pageHost = typeof location === 'undefined' ? '' : location.hostname,
): boolean {
  const endpoint = host.trim();
  if (!endpoint || !isPrivateIPv4(endpoint)) return false;
  return pageHost !== endpoint && pageHost !== 'localhost' && pageHost !== '127.0.0.1';
}
