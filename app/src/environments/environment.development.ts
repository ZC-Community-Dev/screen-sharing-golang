export const environment = {
  production: false,
  /** Proxied by `app/proxy.conf.json` to `http://127.0.0.1:8080`. */
  apiBaseUrl: '/api/v2',
  roomPathPrefix: '/r',
  appOrigin: '',
  allowedMediaTransports: ['webrtc', 'websocket'] as const,
  defaultMediaTransport: 'webrtc' as const,
  /** Empty on localhost: the Angular proxy and Pion share the same host. */
  mediaUdpHost: '',
  mediaUdpPort: 5000,
  mediaUdpMtu: 1200,
};
