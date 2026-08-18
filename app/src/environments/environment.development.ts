export const environment = {
  production: false,
  /** Proxied by `app/proxy.conf.json` to `http://127.0.0.1:8080`. */
  apiBaseUrl: '/api/v2',
  roomPathPrefix: '/r',
  appOrigin: '',
  allowedMediaTransports: ['webrtc', 'websocket'] as const,
  defaultMediaTransport: 'webrtc' as const,
};
