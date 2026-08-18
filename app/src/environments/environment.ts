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
  defaultMediaTransport: 'webrtc' as const,
};
