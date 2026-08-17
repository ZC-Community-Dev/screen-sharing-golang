import { Component, OnDestroy, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription, firstValueFrom } from 'rxjs';

import { ControlBar } from '../../components/control-bar/control-bar';
import { Stage } from '../../components/stage/stage';
import { LinksService } from '../../services/links.service';
import { PresenterTokenStore } from '../../services/presenter-token.store';
import { RoomEvent, RoomEventsService } from '../../services/room-events.service';
import { WebrtcService } from '../../services/webrtc.service';

@Component({
  selector: 'app-room',
  imports: [Stage, ControlBar],
  templateUrl: './room.html',
})
export class Room implements OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly links = inject(LinksService);
  private readonly tokens = inject(PresenterTokenStore);
  private readonly events = inject(RoomEventsService);
  private readonly webrtc = inject(WebrtcService);

  readonly id = signal('');
  readonly role = signal<'presenter' | 'viewer' | ''>('');
  readonly state = signal<'waiting' | 'sharing'>('waiting');
  readonly participantCount = signal(0);
  readonly stream = signal<MediaStream | null>(null);
  readonly publicUrl = signal('');

  private sessionId = '';
  private presenterToken = '';
  private sub?: Subscription;
  private pendingViewers = new Set<string>();

  constructor() {
    const id = this.route.snapshot.paramMap.get('id') ?? '';
    void this.enter(id);
  }

  ngOnDestroy() {
    this.sub?.unsubscribe();
    this.webrtc.stopCapture();
  }

  get canPresent() {
    return this.role() === 'presenter';
  }

  async copyLink() {
    await navigator.clipboard.writeText(`${window.location.origin}${this.publicUrl()}`);
  }

  async startShare() {
    if (!this.canPresent) {
      return;
    }
    const media = await this.webrtc.startCapture();
    this.stream.set(media);
    await firstValueFrom(this.links.startShare(this.id(), this.sessionId, this.presenterToken));
    this.state.set('sharing');
    for (const viewerId of this.pendingViewers) {
      await this.webrtc.createOfferFor(viewerId, (msg) => this.events.send(msg));
    }
    this.pendingViewers.clear();
  }

  async stopShare() {
    this.webrtc.stopCapture();
    this.stream.set(null);
    this.pendingViewers.clear();
    await firstValueFrom(this.links.stopShare(this.id(), this.sessionId, this.presenterToken));
    this.state.set('waiting');
  }

  private async enter(id: string) {
    try {
      const link = await firstValueFrom(this.links.get(id));
      this.id.set(link.id);
      this.state.set(link.state);
      this.applyParticipantCount(link.participantCount);
      this.publicUrl.set(`/r/${link.id}`);
      const token = this.tokens.get(id);
      if (token) {
        const session = await firstValueFrom(this.links.claimPresenter(id, token));
        this.role.set('presenter');
        this.sessionId = session.sessionId;
        this.presenterToken = token;
      } else {
        const session = await firstValueFrom(this.links.joinViewer(id));
        this.role.set('viewer');
        this.sessionId = session.sessionId;
      }
      this.listen();
    } catch {
      await this.router.navigateByUrl('/r/invalid');
    }
  }

  private listen() {
    this.sub = this.events.connect(this.id(), this.sessionId).subscribe((event) => {
      void this.onEvent(event);
    });
    void this.events.whenOpen().then(() => this.sendReadyIfViewerSharing());
  }

  private sendReadyIfViewerSharing() {
    if (this.role() !== 'viewer' || this.state() !== 'sharing') {
      return;
    }
    this.events.send({
      type: 'signal',
      to: 'presenter',
      payload: { kind: 'ready' },
    });
  }

  private applyParticipantCount(value: unknown) {
    const n = Number(value);
    if (Number.isFinite(n) && n >= 0) {
      this.participantCount.set(n);
    }
  }

  private async onEvent(event: RoomEvent) {
    if (event.type === 'room.state') {
      const next = event.payload['state'] === 'sharing' ? 'sharing' : 'waiting';
      this.state.set(next);
      if (next === 'waiting' && this.role() === 'viewer') {
        this.stream.set(null);
        this.webrtc.stopCapture();
      }
      if (next === 'sharing') {
        this.sendReadyIfViewerSharing();
      }
    }
    if (event.type === 'presence') {
      this.applyParticipantCount(event.payload['participantCount']);
    }
    if (event.type !== 'signal' || !event.from) {
      return;
    }
    const payload = event.payload as {
      kind?: string;
      sdp?: string;
      candidate?: RTCIceCandidateInit;
    };
    if (this.role() === 'presenter' && payload.kind === 'ready') {
      if (!this.webrtc.hasLocal()) {
        this.pendingViewers.add(event.from);
        return;
      }
      await this.webrtc.createOfferFor(event.from, (msg) => this.events.send(msg));
      return;
    }
    await this.webrtc.handleSignal(
      event.from,
      payload,
      (msg) => this.events.send(msg),
      (remote) => this.stream.set(remote),
    );
  }
}
