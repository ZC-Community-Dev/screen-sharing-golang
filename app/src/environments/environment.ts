export const environment = {
  production: true,
  /** HTTP JSON prefix. Same-origin (`/api/v1`) or absolute (`https://api.example.com/api/v1`). */
  apiBaseUrl: '/api/v1',
  /** Public room path used when copying the invite (`/r/{id}`). */
  roomPathPrefix: '/r',
  /** STUN servers for mesh WebRTC. MUST NOT include TURN credentials here. */
  stunUrls: ['stun:stun.l.google.com:19302'],
  /** Origin used when copying the public link. Empty = `location.origin`. */
  appOrigin: '',
};
