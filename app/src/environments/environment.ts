export const environment = {
  production: true,
  /** HTTP JSON prefix. Same-origin (`/api/v2`) or absolute (`https://api.example.com/api/v2`). */
  apiBaseUrl: '/api/v2',
  /** Public room path used when copying the invite (`/r/{id}`). */
  roomPathPrefix: '/r',
  /** Origin used when copying the public link. Empty = `location.origin`. */
  appOrigin: '',
  /** Deployment policy; the backend and browser capability checks remain authoritative. */
  allowedMediaTransports: ['webrtc', 'websocket'] as const,
  defaultMediaTransport: 'websocket' as const,
  /**
   * IPv4 reached by browsers for WebRTC/UDP (ICE host). Empty keeps the
   * server SDP unchanged. Set this to the public or LAN address forwarded
   * to `MEDIA_UDP_PORT` when Pion advertises an unreachable interface.
   * Private addresses are unreachable through Cloudflare; internet viewers
   * should use the WebSocket transport instead.
   */
  mediaUdpHost: '',
  /** UDP port of the Go SFU. Must match `MEDIA_UDP_PORT` on the server. */
  mediaUdpPort: 5000,
  /**
   * Max UDP datagram size for WebRTC/ICE (bytes). 1200 avoids
   * fragmentation on typical internet/VPN paths. Range 576–1200.
   */
  mediaUdpMtu: 1200,
};
