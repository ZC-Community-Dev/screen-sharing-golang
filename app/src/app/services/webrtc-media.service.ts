import { Injectable, inject } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { LinksService, SessionDescription } from './links.service';
import { MediaConnectionState } from './media-transport';

type MediaRole = 'publisher' | 'subscriber';
interface ActiveSession {
  role: MediaRole;
  linkId: string;
  sessionId: string;
  presenterToken?: string;
  onRemote?: (stream: MediaStream) => void;
  onState?: (state: MediaConnectionState) => void;
}

const RECONNECT_DELAYS_MS = [500, 1000, 2000] as const;

@Injectable({ providedIn: 'root' })
export class WebRtcMediaService {
  private readonly links = inject(LinksService);
  private peer?: RTCPeerConnection;
  private local?: MediaStream;
  private active?: ActiveSession;
  private mediaSessionId = '';
  private reconnectAttempt = 0;
  private reconnectTimer?: ReturnType<typeof setTimeout>;
  private generation = 0;

  async publish(
    stream: MediaStream,
    linkId: string,
    sessionId: string,
    presenterToken: string,
    onState?: (state: MediaConnectionState) => void,
  ): Promise<void> {
    await this.reset(false);
    this.local = stream;
    this.active = { role: 'publisher', linkId, sessionId, presenterToken, onState };
    await this.negotiate('connecting');
  }

  async subscribe(
    linkId: string,
    sessionId: string,
    onRemote: (stream: MediaStream) => void,
    onState?: (state: MediaConnectionState) => void,
  ): Promise<void> {
    if (
      this.active?.role === 'subscriber' &&
      this.active.linkId === linkId &&
      this.active.sessionId === sessionId &&
      this.peer?.connectionState !== 'closed'
    ) {
      return;
    }
    await this.reset(false);
    this.active = { role: 'subscriber', linkId, sessionId, onRemote, onState };
    await this.negotiate('connecting');
  }

  async stop(): Promise<void> {
    await this.reset(false);
    this.local = undefined;
  }

  private async negotiate(state: 'connecting' | 'reconnecting'): Promise<void> {
    const active = this.active;
    if (!active) return;
    const generation = this.generation;
    active.onState?.(state);
    const peer = new RTCPeerConnection();
    this.peer = peer;
    peer.onconnectionstatechange = () => {
      if (this.peer !== peer || generation !== this.generation) return;
      if (peer.connectionState === 'connected') {
        this.reconnectAttempt = 0;
        active.onState?.('connected');
      } else if (peer.connectionState === 'failed' || peer.connectionState === 'disconnected') {
        this.scheduleReconnect();
      }
    };

    if (active.role === 'publisher') {
      const track = this.local?.getVideoTracks()[0];
      if (!track || !this.local) throw new Error('A captura de tela foi encerrada.');
      const transceiver = peer.addTransceiver(track, {
        direction: 'sendonly',
        streams: [this.local],
      });
      this.preferVp8(transceiver);
    } else {
      const transceiver = peer.addTransceiver('video', { direction: 'recvonly' });
      this.preferVp8(transceiver);
      peer.ontrack = (event) => {
        if (event.track.kind === 'video') {
          active.onRemote?.(event.streams[0] ?? new MediaStream([event.track]));
        }
      };
    }

    try {
      const offer = await peer.createOffer();
      await peer.setLocalDescription(offer);
      await this.waitForIceGathering(peer);
      if (this.peer !== peer || generation !== this.generation || !this.active) {
        peer.close();
        return;
      }
      const completeOffer = this.description(peer.localDescription);
      const response =
        active.role === 'publisher'
          ? await firstValueFrom(
              this.links.createPublisher(active.linkId, {
                sessionId: active.sessionId,
                presenterToken: active.presenterToken!,
                offer: completeOffer,
              }),
            )
          : await firstValueFrom(
              this.links.createSubscriber(active.linkId, {
                sessionId: active.sessionId,
                offer: completeOffer,
              }),
            );
      this.mediaSessionId = response.mediaSessionId;
      await peer.setRemoteDescription(response.answer);
    } catch (error) {
      if (this.peer === peer && generation === this.generation) {
        peer.close();
        this.scheduleReconnect();
      }
      throw error;
    }
  }

  private scheduleReconnect() {
    if (!this.active || this.reconnectTimer) return;
    this.peer?.close();
    this.peer = undefined;
    const delay = RECONNECT_DELAYS_MS[this.reconnectAttempt];
    if (delay === undefined) {
      this.active.onState?.('failed');
      return;
    }
    this.reconnectAttempt += 1;
    this.active.onState?.('reconnecting');
    const generation = this.generation;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = undefined;
      if (!this.active || generation !== this.generation) return;
      void this.deleteSubscriber().finally(() => {
        if (this.active && generation === this.generation) {
          void this.negotiate('reconnecting').catch(() => undefined);
        }
      });
    }, delay);
  }

  private async reset(notify: boolean) {
    this.generation += 1;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = undefined;
    await this.deleteSubscriber();
    this.peer?.close();
    this.peer = undefined;
    if (notify) this.active?.onState?.('closed');
    this.active = undefined;
    this.reconnectAttempt = 0;
  }

  private async deleteSubscriber() {
    if (this.active?.role !== 'subscriber' || !this.mediaSessionId) {
      this.mediaSessionId = '';
      return;
    }
    const { linkId, sessionId } = this.active;
    const mediaSessionId = this.mediaSessionId;
    this.mediaSessionId = '';
    try {
      await firstValueFrom(this.links.deleteSubscriber(linkId, mediaSessionId, sessionId));
    } catch {
      // Connection callbacks make server teardown idempotent.
    }
  }

  private description(description: RTCSessionDescription | null): SessionDescription {
    if (!description?.sdp) throw new Error('A oferta de mídia não foi gerada.');
    return { type: 'offer', sdp: description.sdp };
  }

  private waitForIceGathering(peer: RTCPeerConnection): Promise<void> {
    if (peer.iceGatheringState === 'complete') return Promise.resolve();
    return new Promise((resolve) => {
      peer.onicegatheringstatechange = () => {
        if (peer.iceGatheringState === 'complete') resolve();
      };
    });
  }

  private preferVp8(transceiver: RTCRtpTransceiver) {
    const capabilities =
      typeof RTCRtpSender === 'undefined' ? undefined : RTCRtpSender.getCapabilities?.('video');
    const vp8 = capabilities?.codecs.filter((codec) => codec.mimeType.toLowerCase() === 'video/vp8');
    if (vp8?.length && transceiver.setCodecPreferences) transceiver.setCodecPreferences(vp8);
  }
}
