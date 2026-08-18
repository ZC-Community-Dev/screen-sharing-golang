import { Injectable } from '@angular/core';

import { iceServers } from '../config';

@Injectable({ providedIn: 'root' })
export class WebrtcService {
  private local?: MediaStream;
  private peers = new Map<string, RTCPeerConnection>();
  private iceQueue = new Map<string, RTCIceCandidateInit[]>();

  hasLocal() {
    return !!this.local;
  }

  async startCapture(): Promise<MediaStream> {
    const stream = await navigator.mediaDevices.getDisplayMedia({
      video: true,
      audio: false,
    });
    for (const track of stream.getAudioTracks()) {
      track.stop();
      stream.removeTrack(track);
    }
    this.local = stream;
    return stream;
  }

  stopCapture() {
    this.local?.getTracks().forEach((track) => track.stop());
    this.local = undefined;
    for (const peer of this.peers.values()) {
      peer.close();
    }
    this.peers.clear();
    this.iceQueue.clear();
  }

  async createOfferFor(
    viewerId: string,
    send: (msg: unknown) => void,
  ): Promise<void> {
    if (!this.local || this.peers.has(viewerId)) {
      return;
    }
    const peer = this.peer(viewerId, send);
    this.local.getVideoTracks().forEach((track) => peer.addTrack(track, this.local!));
    const offer = await peer.createOffer();
    await peer.setLocalDescription(offer);
    send({
      type: 'signal',
      to: viewerId,
      payload: { kind: 'offer', sdp: offer.sdp },
    });
  }

  async handleSignal(
    from: string,
    payload: { kind?: string; sdp?: string; candidate?: RTCIceCandidateInit },
    send: (msg: unknown) => void,
    onRemote: (stream: MediaStream) => void,
  ) {
    const peer = this.peer(from, send, onRemote);
    if (payload.kind === 'offer' && payload.sdp) {
      await peer.setRemoteDescription({ type: 'offer', sdp: payload.sdp });
      await this.flushIce(from, peer);
      const answer = await peer.createAnswer();
      await peer.setLocalDescription(answer);
      send({
        type: 'signal',
        to: from,
        payload: { kind: 'answer', sdp: answer.sdp },
      });
    }
    if (payload.kind === 'answer' && payload.sdp) {
      await peer.setRemoteDescription({ type: 'answer', sdp: payload.sdp });
      await this.flushIce(from, peer);
    }
    if (payload.kind === 'ice' && payload.candidate) {
      if (!peer.remoteDescription) {
        const queued = this.iceQueue.get(from) ?? [];
        queued.push(payload.candidate);
        this.iceQueue.set(from, queued);
        return;
      }
      await peer.addIceCandidate(payload.candidate);
    }
  }

  private async flushIce(id: string, peer: RTCPeerConnection) {
    const queued = this.iceQueue.get(id) ?? [];
    this.iceQueue.delete(id);
    for (const candidate of queued) {
      await peer.addIceCandidate(candidate);
    }
  }

  private peer(
    id: string,
    send: (msg: unknown) => void,
    onRemote?: (stream: MediaStream) => void,
  ): RTCPeerConnection {
    const existing = this.peers.get(id);
    if (existing) {
      return existing;
    }
    const peer = new RTCPeerConnection({
      iceServers: iceServers(),
    });
    peer.onicecandidate = (event) => {
      if (event.candidate) {
        send({
          type: 'signal',
          to: id,
          payload: { kind: 'ice', candidate: event.candidate.toJSON() },
        });
      }
    };
    if (onRemote) {
      peer.ontrack = (event) => {
        const stream = event.streams[0] ?? new MediaStream([event.track]);
        onRemote(stream);
      };
    }
    this.peers.set(id, peer);
    return peer;
  }
}
