import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { environment } from '../../environments/environment';
import { apiPath } from '../config';

export type MediaTransport = 'webrtc' | 'websocket';
export type MediaConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'failed' | 'closed';
export type MediaPlayback =
  | { kind: 'stream'; stream: MediaStream }
  | { kind: 'url'; url: string };

export interface WebSocketMediaConfig {
  mimeType: 'video/webm;codecs=vp8';
  timesliceMs: number;
  startupBufferMs: number;
  maxChunkBytes: number;
}

export interface PublicMediaConfig {
  allowedTransports: readonly MediaTransport[];
  defaultTransport: MediaTransport;
  websocket: WebSocketMediaConfig;
}

export interface MediaEnvironment {
  allowedMediaTransports: readonly MediaTransport[];
  defaultMediaTransport: MediaTransport;
}

const TRANSPORTS: readonly MediaTransport[] = ['webrtc', 'websocket'];
export const WEBSOCKET_RECORDER_MIME = 'video/webm;codecs=vp8';
export const WEBSOCKET_MSE_MIME = 'video/webm;codecs="vp8"';

export function validateMediaEnvironment(
  allowedMediaTransports: readonly MediaTransport[],
  defaultMediaTransport: MediaTransport,
): MediaEnvironment {
  if (
    !allowedMediaTransports.length ||
    new Set(allowedMediaTransports).size !== allowedMediaTransports.length ||
    allowedMediaTransports.some((value) => !TRANSPORTS.includes(value)) ||
    !allowedMediaTransports.includes(defaultMediaTransport)
  ) {
    throw new Error('Configuração de transporte de mídia inválida.');
  }
  return { allowedMediaTransports: [...allowedMediaTransports], defaultMediaTransport };
}

interface CapabilityGlobals {
  RTCPeerConnection?: unknown;
  MediaRecorder?: { isTypeSupported(type: string): boolean };
  MediaSource?: { isTypeSupported(type: string): boolean };
}

export function runtimeMediaCapabilities(
  userAgent = globalThis.navigator?.userAgent ?? '',
  globals: CapabilityGlobals = globalThis as CapabilityGlobals,
): readonly MediaTransport[] {
  const result: MediaTransport[] = [];
  if (typeof globals.RTCPeerConnection !== 'undefined') {
    result.push('webrtc');
  }
  const chromiumVersion = /\b(?:Chrome|Chromium|Edg)\/(\d+)/i.exec(userAgent);
  const chromium =
    Number(chromiumVersion?.[1]) >= 120 &&
    !/\b(?:Firefox|OPR|SamsungBrowser)\//i.test(userAgent);
  if (
    chromium &&
    globals.MediaRecorder?.isTypeSupported(WEBSOCKET_RECORDER_MIME) &&
    globals.MediaSource?.isTypeSupported(WEBSOCKET_MSE_MIME)
  ) {
    result.push('websocket');
  }
  return result;
}

@Injectable({ providedIn: 'root' })
export class MediaTransportService {
  private readonly http = inject(HttpClient);
  private readonly deployment = validateMediaEnvironment(
    environment.allowedMediaTransports,
    environment.defaultMediaTransport,
  );
  private server?: PublicMediaConfig;
  private availableValue: readonly MediaTransport[] = [];

  get available(): readonly MediaTransport[] {
    return this.availableValue;
  }

  get defaultTransport(): MediaTransport | null {
    if (this.availableValue.includes(this.deployment.defaultMediaTransport)) {
      return this.deployment.defaultMediaTransport;
    }
    const serverDefault = this.server?.defaultTransport;
    return serverDefault && this.availableValue.includes(serverDefault) ? serverDefault : null;
  }

  get websocketConfig(): WebSocketMediaConfig | undefined {
    return this.server?.websocket;
  }

  async load(): Promise<readonly MediaTransport[]> {
    const server = await firstValueFrom(
      this.http.get<PublicMediaConfig>(apiPath('/media/config')),
    );
    validateMediaEnvironment(server.allowedTransports, server.defaultTransport);
    this.server = server;
    const runtime = runtimeMediaCapabilities();
    this.availableValue = this.deployment.allowedMediaTransports.filter(
      (transport) => server.allowedTransports.includes(transport) && runtime.includes(transport),
    );
    return this.availableValue;
  }

  requireAvailable(transport: MediaTransport): void {
    if (!this.availableValue.includes(transport)) {
      throw new Error(`O transporte ${transport} não está disponível neste navegador.`);
    }
  }
}
