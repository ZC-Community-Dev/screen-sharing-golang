import { Injectable, inject } from '@angular/core';

import {
  MediaConnectionState,
  MediaPlayback,
  MediaTransport,
  MediaTransportService,
} from './media-transport';
import { WebRtcMediaService } from './webrtc-media.service';
import { WebSocketMediaService } from './websocket-media.service';

export type { MediaConnectionState, MediaPlayback, MediaTransport };

@Injectable({ providedIn: 'root' })
export class MediaService {
  private readonly transports = inject(MediaTransportService);
  private readonly webrtc = inject(WebRtcMediaService);
  private readonly websocket = inject(WebSocketMediaService);
  private local?: MediaStream;
  private activeTransport?: MediaTransport;
  private configured = false;

  get availableTransports(): readonly MediaTransport[] {
    return this.transports.available;
  }

  get defaultTransport(): MediaTransport | null {
    return this.transports.defaultTransport;
  }

  async loadTransports(): Promise<readonly MediaTransport[]> {
    const available = await this.transports.load();
    this.configured = true;
    return available;
  }

  async publish(
    linkId: string,
    sessionId: string,
    presenterToken: string,
    onState?: (state: MediaConnectionState) => void,
    transport: MediaTransport = 'webrtc',
  ): Promise<MediaStream> {
    await this.stop();
    this.assertTransport(transport);
    const stream = await navigator.mediaDevices.getDisplayMedia({
      video: { frameRate: { max: 15 }, width: { max: 1920 }, height: { max: 1080 } },
      audio: false,
    });
    for (const track of stream.getAudioTracks()) {
      track.stop();
      stream.removeTrack(track);
    }
    const videoTrack = stream.getVideoTracks()[0];
    if (!videoTrack) {
      stream.getTracks().forEach((track) => track.stop());
      throw new Error('A captura de tela não forneceu vídeo.');
    }

    this.local = stream;
    this.activeTransport = transport;
    videoTrack.addEventListener?.('ended', () => void this.stop());
    try {
      if (transport === 'webrtc') {
        await this.webrtc.publish(stream, linkId, sessionId, presenterToken, onState);
      } else {
        const config = this.transports.websocketConfig;
        if (!config) throw new Error('Configuração WebSocket indisponível.');
        await this.websocket.publish(stream, linkId, sessionId, presenterToken, config, onState);
      }
      return stream;
    } catch (error) {
      stream.getTracks().forEach((track) => track.stop());
      this.local = undefined;
      this.activeTransport = undefined;
      throw error;
    }
  }

  async subscribe(
    linkId: string,
    sessionId: string,
    onRemote: (playback: MediaStream | MediaPlayback) => void,
    onState?: (state: MediaConnectionState) => void,
    transport: MediaTransport = 'webrtc',
  ): Promise<void> {
    if (this.activeTransport && this.activeTransport !== transport) {
      throw new Error('O transporte da publicação não corresponde à sessão ativa.');
    }
    this.assertTransport(transport);
    this.activeTransport = transport;
    if (transport === 'webrtc') {
      await this.webrtc.subscribe(
        linkId,
        sessionId,
        onRemote,
        onState,
      );
    } else {
      const config = this.transports.websocketConfig;
      if (!config) throw new Error('Configuração WebSocket indisponível.');
      await this.websocket.subscribe(linkId, sessionId, config, onRemote, onState);
    }
  }

  async stop(): Promise<void> {
    await Promise.all([this.webrtc.stop(), this.websocket.stop()]);
    this.local?.getTracks().forEach((track) => track.stop());
    this.local = undefined;
    this.activeTransport = undefined;
  }

  private assertTransport(transport: MediaTransport) {
    if (this.configured) this.transports.requireAvailable(transport);
  }
}
