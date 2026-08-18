import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import { LinksService } from './links.service';
import { WebSocketMediaConfig } from './media-transport';
import { WebSocketMediaService } from './websocket-media.service';

class FakeSocket {
  static instances: FakeSocket[] = [];
  static readonly OPEN = 1;
  static readonly CLOSED = 3;
  readyState = 1;
  bufferedAmount = 0;
  binaryType = '';
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn();
  constructor(
    readonly url: string,
    readonly protocol: string,
  ) {
    FakeSocket.instances.push(this);
  }
}

class FakeRecorder {
  static instances: FakeRecorder[] = [];
  state: RecordingState = 'inactive';
  ondataavailable: ((event: BlobEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  start = vi.fn((timeslice: number) => {
    this.state = 'recording';
    void timeslice;
  });
  stop = vi.fn(() => (this.state = 'inactive'));
  constructor(
    readonly stream: MediaStream,
    readonly options: MediaRecorderOptions,
  ) {
    FakeRecorder.instances.push(this);
  }
  static isTypeSupported() {
    return true;
  }
}

class FakeSourceBuffer {
  updating = false;
  mode: AppendMode = 'segments';
  readonly appended: ArrayBuffer[] = [];
  readonly buffered = {
    length: 0,
    start: () => 0,
    end: () => 0,
  } as TimeRanges;
  private updateEnd?: () => void;
  addEventListener(_type: string, listener: EventListenerOrEventListenerObject) {
    this.updateEnd =
      typeof listener === 'function' ? () => listener(new Event('updateend')) : undefined;
  }
  appendBuffer(data: BufferSource) {
    this.updating = true;
    this.appended.push(data as ArrayBuffer);
  }
  remove = vi.fn();
  finishAppend() {
    this.updating = false;
    this.updateEnd?.();
  }
}

class FakeMediaSource {
  static instances: FakeMediaSource[] = [];
  readyState: ReadyState = 'open';
  readonly buffer = new FakeSourceBuffer();
  private sourceOpen?: () => void;
  constructor() {
    FakeMediaSource.instances.push(this);
  }
  static isTypeSupported() {
    return true;
  }
  addEventListener(
    _type: string,
    listener: EventListenerOrEventListenerObject,
    _options?: AddEventListenerOptions,
  ) {
    this.sourceOpen =
      typeof listener === 'function' ? () => listener(new Event('sourceopen')) : undefined;
  }
  addSourceBuffer = vi.fn(() => this.buffer);
  endOfStream = vi.fn();
  open() {
    this.sourceOpen?.();
  }
}

const config: WebSocketMediaConfig = {
  mimeType: 'video/webm;codecs=vp8',
  timesliceMs: 250,
  startupBufferMs: 2000,
  maxChunkBytes: 1024,
};

describe('WebSocketMediaService', () => {
  const links = {
    createWebSocketTicket: vi.fn(() =>
      of({
        ticket: 'one-use-ticket',
        expiresAt: new Date(Date.now() + 30_000).toISOString(),
        websocketPath: '/api/v2/links/abc/media/websocket',
      }),
    ),
  };
  const videoTrack = { kind: 'video' } as MediaStreamTrack;
  const display = { getVideoTracks: () => [videoTrack] } as unknown as MediaStream;

  beforeEach(() => {
    FakeSocket.instances = [];
    FakeRecorder.instances = [];
    FakeMediaSource.instances = [];
    vi.clearAllMocks();
    vi.stubGlobal('WebSocket', FakeSocket);
    vi.stubGlobal('MediaRecorder', FakeRecorder);
    vi.stubGlobal(
      'MediaStream',
      class {
        constructor(readonly tracks: MediaStreamTrack[]) {}
      },
    );
    TestBed.configureTestingModule({
      providers: [{ provide: LinksService, useValue: links }],
    });
  });

  it('publishes ordered video-only VP8 Blobs at 250ms with bounded backpressure', async () => {
    const service = TestBed.inject(WebSocketMediaService);
    const pending = service.publish(display, 'abc', 'presenter-1', 'secret', config);
    await Promise.resolve();
    await Promise.resolve();
    const socket = FakeSocket.instances[0];
    expect(socket.protocol).toBe('screen-share-media-v1');
    expect(socket.url).toContain('ticket=one-use-ticket');
    socket.onopen?.();
    expect(JSON.parse(socket.send.mock.calls[0][0])).toMatchObject({
      type: 'publisher.open',
      mimeType: 'video/webm;codecs=vp8',
      timesliceMs: 250,
    });
    socket.onmessage?.({
      data: JSON.stringify({ type: 'media.opened', publicationId: 'p1', mediaSessionId: 'm1' }),
    } as MessageEvent);
    await pending;

    const recorder = FakeRecorder.instances[0];
    expect(recorder.options).toMatchObject({
      mimeType: 'video/webm;codecs=vp8',
      videoBitsPerSecond: 2_500_000,
    });
    expect(recorder.start).toHaveBeenCalledWith(250);
    const first = new Blob(['first']);
    const second = new Blob(['second']);
    recorder.ondataavailable?.({ data: first } as BlobEvent);
    recorder.ondataavailable?.({ data: second } as BlobEvent);
    expect(socket.send.mock.calls.slice(-2).map((call) => call[0])).toEqual([first, second]);

    socket.bufferedAmount = 4096;
    recorder.ondataavailable?.({ data: new Blob(['blocked']) } as BlobEvent);
    expect(socket.send).not.toHaveBeenCalledWith(expect.objectContaining({ size: 7 }));
  });

  it('reconnects with a fresh ticket on the same transport and cancels on stop', async () => {
    vi.useFakeTimers();
    const service = TestBed.inject(WebSocketMediaService);
    const pending = service.publish(display, 'abc', 'presenter-1', 'secret', config);
    await Promise.resolve();
    await Promise.resolve();
    const first = FakeSocket.instances[0];
    first.onopen?.();
    first.onmessage?.({
      data: JSON.stringify({ type: 'media.opened', publicationId: 'p1', mediaSessionId: 'm1' }),
    } as MessageEvent);
    await pending;
    first.onclose?.();
    await vi.advanceTimersByTimeAsync(500);
    expect(links.createWebSocketTicket).toHaveBeenCalledTimes(2);
    expect(FakeSocket.instances).toHaveLength(2);
    await service.stop();
    await vi.runAllTimersAsync();
    expect(FakeSocket.instances).toHaveLength(2);
    vi.useRealTimers();
  });

  it('serializes MediaSource appends, resets playback, and revokes object URLs', async () => {
    vi.stubGlobal('MediaSource', FakeMediaSource);
    const createObjectURL = vi.fn(() => `blob:media-${FakeMediaSource.instances.length}`);
    const revokeObjectURL = vi.fn();
    Object.defineProperties(URL, {
      createObjectURL: { configurable: true, value: createObjectURL },
      revokeObjectURL: { configurable: true, value: revokeObjectURL },
    });
    const service = TestBed.inject(WebSocketMediaService);
    const playback = vi.fn();
    const pending = service.subscribe('abc', 'viewer-1', config, playback);
    await Promise.resolve();
    await Promise.resolve();
    const source = FakeMediaSource.instances[0];
    source.open();
    const socket = FakeSocket.instances[0];
    socket.onopen?.();
    socket.onmessage?.({
      data: JSON.stringify({ type: 'media.opened', publicationId: 'p1', mediaSessionId: 'm1' }),
    } as MessageEvent);
    await pending;

    const first = new Uint8Array([1]).buffer;
    const second = new Uint8Array([2]).buffer;
    socket.onmessage?.({ data: first } as MessageEvent);
    socket.onmessage?.({ data: second } as MessageEvent);
    expect(source.buffer.appended).toEqual([first]);
    source.buffer.finishAppend();
    expect(source.buffer.appended).toEqual([first, second]);

    socket.onmessage?.({ data: JSON.stringify({ type: 'media.reset', generation: 2 }) } as MessageEvent);
    expect(playback).toHaveBeenLastCalledWith({ kind: 'url', url: 'blob:media-2' });
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:media-1');
    await service.stop();
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:media-2');
  });
});
