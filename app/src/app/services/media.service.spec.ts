import { provideHttpClient } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import { LinksService } from './links.service';
import { MediaService } from './media.service';

class FakePeerConnection {
  static instances: FakePeerConnection[] = [];

  connectionState: RTCPeerConnectionState = 'new';
  iceGatheringState: RTCIceGatheringState = 'new';
  localDescription: RTCSessionDescription | null = null;
  remoteDescription: RTCSessionDescription | null = null;
  onconnectionstatechange: (() => void) | null = null;
  onicegatheringstatechange: (() => void) | null = null;
  ontrack: ((event: RTCTrackEvent) => void) | null = null;
  readonly addTrack = vi.fn();
  readonly addTransceiver = vi.fn(() => ({ setCodecPreferences: vi.fn() }));
  readonly close = vi.fn();

  constructor() {
    FakePeerConnection.instances.push(this);
  }

  async createOffer() {
    return { type: 'offer' as RTCSdpType, sdp: 'complete-offer' };
  }

  async setLocalDescription(description: RTCSessionDescriptionInit) {
    this.localDescription = description as RTCSessionDescription;
    this.iceGatheringState = 'complete';
  }

  async setRemoteDescription(description: RTCSessionDescriptionInit) {
    this.remoteDescription = description as RTCSessionDescription;
  }
}

describe('MediaService', () => {
  const videoTrack = { kind: 'video', stop: vi.fn() } as unknown as MediaStreamTrack;
  const display = {
    getVideoTracks: () => [videoTrack],
    getAudioTracks: () => [],
    getTracks: () => [videoTrack],
  } as unknown as MediaStream;
  const links = {
    createPublisher: vi.fn(() =>
      of({
        mediaSessionId: 'publisher-1',
        answer: { type: 'answer' as const, sdp: 'publisher-answer' },
      }),
    ),
    createSubscriber: vi.fn(() =>
      of({
        mediaSessionId: 'subscriber-1',
        answer: { type: 'answer' as const, sdp: 'subscriber-answer' },
      }),
    ),
    deleteSubscriber: vi.fn(() => of(undefined)),
  };

  beforeEach(() => {
    FakePeerConnection.instances = [];
    vi.clearAllMocks();
    Object.defineProperty(globalThis, 'RTCPeerConnection', {
      configurable: true,
      value: FakePeerConnection,
    });
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: { getDisplayMedia: vi.fn().mockResolvedValue(display) },
    });
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), { provide: LinksService, useValue: links }],
    });
  });

  it('publishes exactly one screen-video track through one sendonly server peer', async () => {
    const service = TestBed.inject(MediaService);

    const stream = await service.publish('link-1', 'room-1', 'secret');

    expect(stream).toBe(display);
    expect(FakePeerConnection.instances).toHaveLength(1);
    const peer = FakePeerConnection.instances[0];
    expect(peer.addTransceiver).toHaveBeenCalledWith(videoTrack, { direction: 'sendonly', streams: [display] });
    expect(links.createPublisher).toHaveBeenCalledWith('link-1', {
      sessionId: 'room-1',
      presenterToken: 'secret',
      offer: { type: 'offer', sdp: 'complete-offer' },
    });
    expect(peer.remoteDescription?.sdp).toBe('publisher-answer');
  });

  it('subscribes with one recvonly peer, forwards its remote stream, and deletes on teardown', async () => {
    const service = TestBed.inject(MediaService);
    const onRemote = vi.fn();

    await service.subscribe('link-1', 'viewer-1', onRemote);

    expect(FakePeerConnection.instances).toHaveLength(1);
    const peer = FakePeerConnection.instances[0];
    expect(peer.addTransceiver).toHaveBeenCalledWith('video', { direction: 'recvonly' });
    peer.ontrack?.({ streams: [display], track: videoTrack } as unknown as RTCTrackEvent);
    expect(onRemote).toHaveBeenCalledWith(display);

    await service.stop();
    expect(links.deleteSubscriber).toHaveBeenCalledWith('link-1', 'subscriber-1', 'viewer-1');
    expect(peer.close).toHaveBeenCalled();
  });

  it('uses bounded reconnect and cancels pending retries on stop', async () => {
    vi.useFakeTimers();
    const service = TestBed.inject(MediaService);
    await service.subscribe('link-1', 'viewer-1', vi.fn());
    const first = FakePeerConnection.instances[0];

    first.connectionState = 'failed';
    first.onconnectionstatechange?.();
    await vi.advanceTimersByTimeAsync(500);
    expect(FakePeerConnection.instances).toHaveLength(2);

    FakePeerConnection.instances[1].connectionState = 'failed';
    FakePeerConnection.instances[1].onconnectionstatechange?.();
    await service.stop();
    await vi.runAllTimersAsync();
    expect(FakePeerConnection.instances).toHaveLength(2);
    vi.useRealTimers();
  });

  it('reports failed after three reconnect attempts', async () => {
    vi.useFakeTimers();
    const service = TestBed.inject(MediaService);
    const onState = vi.fn();
    await service.subscribe('link-1', 'viewer-1', vi.fn(), onState);

    for (const delay of [500, 1000, 2000]) {
      const peer = FakePeerConnection.instances.at(-1)!;
      peer.connectionState = 'failed';
      peer.onconnectionstatechange?.();
      await vi.advanceTimersByTimeAsync(delay);
    }
    const finalPeer = FakePeerConnection.instances.at(-1)!;
    finalPeer.connectionState = 'failed';
    finalPeer.onconnectionstatechange?.();

    expect(FakePeerConnection.instances).toHaveLength(4);
    expect(onState).toHaveBeenLastCalledWith('failed');
    await service.stop();
    vi.useRealTimers();
  });
});
