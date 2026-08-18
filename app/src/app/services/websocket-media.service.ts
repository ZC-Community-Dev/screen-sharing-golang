import { Injectable, inject } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { mediaWsUrl } from '../config';
import { LinksService } from './links.service';
import {
  MediaConnectionState,
  WEBSOCKET_MSE_MIME,
  WEBSOCKET_RECORDER_MIME,
  WebSocketMediaConfig,
} from './media-transport';

export type MediaSourcePlayback = { kind: 'url'; url: string };

interface ActiveWebSocketSession {
  role: 'publisher' | 'viewer';
  linkId: string;
  sessionId: string;
  presenterToken?: string;
  stream?: MediaStream;
  config: WebSocketMediaConfig;
  onPlayback?: (playback: MediaSourcePlayback) => void;
  onState?: (state: MediaConnectionState) => void;
}

type ServerControl =
  | { type: 'media.opened'; publicationId: string; mediaSessionId: string }
  | { type: 'media.bootstrap'; generation: number; clusterCount: number }
  | { type: 'media.live' }
  | { type: 'media.reset'; generation?: number }
  | { type: 'media.end' }
  | { type: 'media.error'; code: string; message?: string };

const RECONNECT_DELAYS_MS = [500, 1000, 2000] as const;
const VIDEO_BITS_PER_SECOND = 2_500_000;

@Injectable({ providedIn: 'root' })
export class WebSocketMediaService {
  private readonly links = inject(LinksService);
  private socket?: WebSocket;
  private recorder?: MediaRecorder;
  private active?: ActiveWebSocketSession;
  private reconnectTimer?: ReturnType<typeof setTimeout>;
  private reconnectAttempt = 0;
  private generation = 0;
  private stopping = false;
  private source?: MediaSource;
  private sourceBuffer?: SourceBuffer;
  private appendQueue: ArrayBuffer[] = [];
  private objectUrl = '';

  async publish(
    stream: MediaStream,
    linkId: string,
    sessionId: string,
    presenterToken: string,
    config: WebSocketMediaConfig,
    onState?: (state: MediaConnectionState) => void,
  ): Promise<void> {
    await this.stop();
    this.stopping = false;
    this.active = { role: 'publisher', stream, linkId, sessionId, presenterToken, config, onState };
    await this.connect('connecting');
  }

  async subscribe(
    linkId: string,
    sessionId: string,
    config: WebSocketMediaConfig,
    onPlayback: (playback: MediaSourcePlayback) => void,
    onState?: (state: MediaConnectionState) => void,
  ): Promise<void> {
    if (
      this.active?.role === 'viewer' &&
      this.active.linkId === linkId &&
      this.active.sessionId === sessionId &&
      this.socket?.readyState !== WebSocket.CLOSED
    ) {
      return;
    }
    await this.stop();
    this.stopping = false;
    this.active = { role: 'viewer', linkId, sessionId, config, onPlayback, onState };
    this.createMediaSource();
    await this.connect('connecting');
  }

  async stop(): Promise<void> {
    this.stopping = true;
    this.generation += 1;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = undefined;
    if (this.recorder?.state !== 'inactive') {
      this.sendControl({ type: 'media.end' });
      this.recorder?.stop();
    }
    this.recorder = undefined;
    const socket = this.socket;
    this.socket = undefined;
    socket?.close(1000, 'media ended');
    this.active?.onState?.('closed');
    this.active = undefined;
    this.reconnectAttempt = 0;
    this.destroyMediaSource();
  }

  private async connect(state: 'connecting' | 'reconnecting'): Promise<void> {
    const active = this.active;
    if (!active) return;
    const generation = ++this.generation;
    active.onState?.(state);
    const ticket = await firstValueFrom(
      this.links.createWebSocketTicket(active.linkId, {
        sessionId: active.sessionId,
        role: active.role,
        ...(active.role === 'publisher' ? { presenterToken: active.presenterToken } : {}),
      }),
    );
    if (!this.active || generation !== this.generation) return;

    await new Promise<void>((resolve, reject) => {
      const socket = new WebSocket(
        mediaWsUrl(ticket.websocketPath, ticket.ticket),
        'screen-share-media-v1',
      );
      socket.binaryType = 'arraybuffer';
      this.socket = socket;
      let opened = false;
      socket.onopen = () => {
        if (active.role === 'publisher') {
          this.sendControl({
            type: 'publisher.open',
            protocolVersion: 1,
            mimeType: WEBSOCKET_RECORDER_MIME,
            timesliceMs: active.config.timesliceMs,
          });
        } else {
          this.sendControl({ type: 'subscriber.open', protocolVersion: 1 });
        }
      };
      socket.onmessage = (event) => {
        if (generation !== this.generation || this.socket !== socket) return;
        if (typeof event.data === 'string') {
          const message = JSON.parse(event.data) as ServerControl;
          if (message.type === 'media.opened') {
            opened = true;
            this.reconnectAttempt = 0;
            if (active.role === 'publisher') this.startRecorder(active);
            resolve();
          } else if (message.type === 'media.live') {
            active.onState?.('connected');
          } else if (message.type === 'media.reset') {
            this.createMediaSource();
          } else if (message.type === 'media.end') {
            active.onState?.('closed');
            void this.stop();
          } else if (message.type === 'media.error') {
            active.onState?.('failed');
          }
          return;
        }
        if (active.role === 'viewer' && event.data instanceof ArrayBuffer) {
          this.enqueue(event.data);
        }
      };
      socket.onerror = () => {
        if (!opened) reject(new Error('Não foi possível abrir o transporte WebSocket.'));
      };
      socket.onclose = () => {
        if (this.socket === socket) this.socket = undefined;
        if (!opened) reject(new Error('O transporte WebSocket foi fechado antes de abrir.'));
        if (!this.stopping && generation === this.generation) this.scheduleReconnect();
      };
    });
  }

  private startRecorder(active: ActiveWebSocketSession) {
    const videoTrack = active.stream?.getVideoTracks()[0];
    if (!videoTrack || !active.stream) throw new Error('A captura de tela foi encerrada.');
    if (this.recorder?.state !== 'inactive') this.recorder?.stop();
    const videoOnly = new MediaStream([videoTrack]);
    const recorder = new MediaRecorder(videoOnly, {
      mimeType: WEBSOCKET_RECORDER_MIME,
      videoBitsPerSecond: VIDEO_BITS_PER_SECOND,
    });
    this.recorder = recorder;
    recorder.ondataavailable = (event) => {
      const socket = this.socket;
      if (
        event.data.size > 0 &&
        event.data.size <= active.config.maxChunkBytes &&
        socket?.readyState === WebSocket.OPEN &&
        socket.bufferedAmount <= active.config.maxChunkBytes * 2
      ) {
        socket.send(event.data);
      } else if (event.data.size > active.config.maxChunkBytes) {
        active.onState?.('failed');
        void this.stop();
      } else if (event.data.size > 0 && socket?.bufferedAmount) {
        this.scheduleReconnect();
      }
    };
    recorder.onerror = () => this.scheduleReconnect();
    recorder.start(active.config.timesliceMs);
    active.onState?.('connected');
  }

  private scheduleReconnect() {
    const active = this.active;
    if (!active || this.reconnectTimer || this.stopping) return;
    const delay = RECONNECT_DELAYS_MS[this.reconnectAttempt];
    if (delay === undefined) {
      active.onState?.('failed');
      return;
    }
    this.reconnectAttempt += 1;
    active.onState?.('reconnecting');
    if (this.recorder?.state !== 'inactive') this.recorder?.stop();
    this.recorder = undefined;
    const generation = this.generation;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = undefined;
      if (!this.active || generation !== this.generation || this.stopping) return;
      if (active.role === 'viewer') this.createMediaSource();
      void this.connect('reconnecting').catch(() => this.scheduleReconnect());
    }, delay);
    const socket = this.socket;
    this.socket = undefined;
    socket?.close(1011, 'media reconnect');
  }

  private createMediaSource() {
    this.destroyMediaSource();
    const source = new MediaSource();
    const url = URL.createObjectURL(source);
    this.source = source;
    this.objectUrl = url;
    this.active?.onPlayback?.({ kind: 'url', url });
    source.addEventListener(
      'sourceopen',
      () => {
        if (this.source !== source) return;
        const buffer = source.addSourceBuffer(WEBSOCKET_MSE_MIME);
        buffer.mode = 'segments';
        buffer.addEventListener('updateend', () => this.flushQueue());
        this.sourceBuffer = buffer;
        this.flushQueue();
      },
      { once: true },
    );
  }

  private enqueue(data: ArrayBuffer) {
    this.appendQueue.push(data.slice(0));
    this.flushQueue();
  }

  private flushQueue() {
    const buffer = this.sourceBuffer;
    if (!buffer || buffer.updating || !this.appendQueue.length) return;
    const next = this.appendQueue[0];
    try {
      buffer.appendBuffer(next);
      this.appendQueue.shift();
    } catch (error) {
      if (error instanceof DOMException && error.name === 'QuotaExceededError' && buffer.buffered.length) {
        const keepFrom = Math.max(0, buffer.buffered.end(buffer.buffered.length - 1) - 2);
        if (keepFrom > buffer.buffered.start(0)) buffer.remove(buffer.buffered.start(0), keepFrom);
      } else {
        this.active?.onState?.('failed');
      }
    }
  }

  private destroyMediaSource() {
    const source = this.source;
    if (source?.readyState === 'open' && !this.sourceBuffer?.updating) {
      try {
        source.endOfStream();
      } catch {
        // Source may already be ending.
      }
    }
    this.sourceBuffer = undefined;
    this.source = undefined;
    this.appendQueue = [];
    if (this.objectUrl) URL.revokeObjectURL(this.objectUrl);
    this.objectUrl = '';
  }

  private sendControl(message: object) {
    if (this.socket?.readyState === WebSocket.OPEN) this.socket.send(JSON.stringify(message));
  }
}
