export const environment = {
  production: false,
  /** Proxied by `app/proxy.conf.json` to `http://127.0.0.1:8080`. */
  apiBaseUrl: '/api/v1',
  roomPathPrefix: '/r',
  stunUrls: ['stun:stun.l.google.com:19302'],
  appOrigin: '',
};
