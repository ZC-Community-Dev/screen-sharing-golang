import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import {
  MediaTransportService,
  runtimeMediaCapabilities,
  validateMediaEnvironment,
} from './media-transport';

describe('media transport configuration', () => {
  it('rejects empty, duplicate, unknown, and non-member defaults', () => {
    expect(() => validateMediaEnvironment([], 'webrtc')).toThrow();
    expect(() => validateMediaEnvironment(['webrtc', 'webrtc'], 'webrtc')).toThrow();
    expect(() => validateMediaEnvironment(['other' as never], 'other' as never)).toThrow();
    expect(() => validateMediaEnvironment(['webrtc'], 'websocket')).toThrow();
  });

  it('offers WebSocket only to validated Chromium with VP8 Recorder and MSE', () => {
    const globals = {
      RTCPeerConnection: class {},
      MediaRecorder: { isTypeSupported: (mime: string) => mime === 'video/webm;codecs=vp8' },
      MediaSource: { isTypeSupported: (mime: string) => mime === 'video/webm;codecs="vp8"' },
    };
    expect(runtimeMediaCapabilities('Mozilla/5.0 Chrome/140.0.0.0 Safari/537.36', globals)).toEqual([
      'webrtc',
      'websocket',
    ]);
    expect(runtimeMediaCapabilities('Mozilla/5.0 Firefox/141.0', globals)).toEqual(['webrtc']);
  });

  it('intersects deployment, server, and runtime without fallback', async () => {
    vi.stubGlobal('RTCPeerConnection', class {});
    vi.stubGlobal('MediaRecorder', {
      isTypeSupported: (mime: string) => mime === 'video/webm;codecs=vp8',
    });
    vi.stubGlobal('MediaSource', {
      isTypeSupported: (mime: string) => mime === 'video/webm;codecs="vp8"',
    });
    Object.defineProperty(navigator, 'userAgent', {
      configurable: true,
      value: 'Mozilla/5.0 Chrome/140.0.0.0 Safari/537.36',
    });
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    const service = TestBed.inject(MediaTransportService);
    const pending = service.load();
    TestBed.inject(HttpTestingController).expectOne('/api/v2/media/config').flush({
      allowedTransports: ['websocket'],
      defaultTransport: 'websocket',
      websocket: {
        mimeType: 'video/webm;codecs=vp8',
        timesliceMs: 250,
        startupBufferMs: 2000,
        maxChunkBytes: 4194304,
      },
    });
    await expect(pending).resolves.toEqual(['websocket']);
    expect(service.defaultTransport).toBe('websocket');
    expect(() => service.requireAvailable('webrtc')).toThrow();
  });
});
