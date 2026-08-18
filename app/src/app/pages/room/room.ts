import { Component, OnDestroy, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription, firstValueFrom } from 'rxjs';

import { publicOrigin, roomPath, webrtcUdpMayBeUnreachable } from '../../config';
import { environment } from '../../../environments/environment';
import { ControlBar } from '../../components/control-bar/control-bar';
import { MediaTransportSelector } from '../../components/media-transport-selector/media-transport-selector';
import { Stage } from '../../components/stage/stage';
import { LinksService, PublicationPublic, RoomState } from '../../services/links.service';
import {
  MediaConnectionState,
  MediaPlayback,
  MediaService,
  MediaTransport,
} from '../../services/media.service';
import { PresenterTokenStore } from '../../services/presenter-token.store';
import { RoomEvent, RoomEventsService } from '../../services/room-events.service';

@Component({
  selector: 'app-room',
  imports: [Stage, ControlBar, MediaTransportSelector],
  templateUrl: './room.html',
})
export class Room implements OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly links = inject(LinksService);
  private readonly tokens = inject(PresenterTokenStore);
  private readonly events = inject(RoomEventsService);
  private readonly media = inject(MediaService);

  readonly id = signal('');
  readonly role = signal<'presenter' | 'viewer' | ''>('');
  readonly state = signal<RoomState>('waiting');
  readonly participantCount = signal(0);
  readonly playback = signal<MediaPlayback | null>(null);
  readonly publicUrl = signal('');
  readonly mediaTransports = signal<readonly MediaTransport[]>([]);
  readonly selectedTransport = signal<MediaTransport>(environment.defaultMediaTransport);
  readonly mediaError = signal('');
  readonly webrtcUdpMayBeUnreachable = webrtcUdpMayBeUnreachable;

  private sessionId = '';
  private presenterToken = '';
  private publication?: PublicationPublic;
  private sub?: Subscription;

  constructor() {
    const id = this.route.snapshot.paramMap.get('id') ?? '';
    void this.enter(id);
  }

  ngOnDestroy() {
    this.sub?.unsubscribe();
    void this.media.stop();
  }

  get canPresent() {
    return this.role() === 'presenter';
  }

  get showTransportSelector() {
    return (
      this.canPresent &&
      (this.state() === 'waiting' || this.state() === 'failed') &&
      this.mediaTransports().length > 1
    );
  }

  async copyLink() {
    await navigator.clipboard.writeText(`${publicOrigin()}${this.publicUrl()}`);
  }

  async startShare() {
    if (!this.canPresent) {
      return;
    }
    const transport = this.selectedTransport();
    if (!this.mediaTransports().includes(transport)) {
      this.mediaError.set('O transporte selecionado não está disponível.');
      this.state.set('failed');
      return;
    }
    this.mediaError.set('');
    this.state.set('connecting');
    try {
      const stream = await this.media.publish(
        this.id(),
        this.sessionId,
        this.presenterToken,
        (state) => this.onMediaState(state),
        transport,
      );
      this.playback.set({ kind: 'stream', stream });
    } catch (error) {
      this.playback.set(null);
      this.mediaError.set(error instanceof Error ? error.message : 'Falha ao iniciar a transmissão.');
      this.state.set('failed');
    }
  }

  async stopShare() {
    await this.media.stop();
    this.playback.set(null);
    this.publication = undefined;
    await firstValueFrom(this.links.stopShare(this.id(), this.sessionId, this.presenterToken));
    this.state.set('waiting');
  }

  private async enter(id: string) {
    try {
      const link = await firstValueFrom(this.links.get(id));
      this.id.set(link.id);
      this.state.set(link.state);
      this.applyParticipantCount(link.participantCount);
      this.publicUrl.set(roomPath(link.id));
      this.publication = link.publication ?? undefined;
      try {
        const available = await this.media.loadTransports();
        this.mediaTransports.set(available);
        const defaultTransport = this.media.defaultTransport ?? available[0];
        if (defaultTransport) this.selectedTransport.set(defaultTransport);
        if (!available.length) {
          this.mediaError.set('Nenhum transporte de mídia está disponível neste navegador.');
          this.state.set('failed');
        }
      } catch {
        this.mediaError.set('Não foi possível carregar a configuração de mídia.');
        this.state.set('failed');
      }
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
      if (this.role() === 'viewer' && this.state() === 'sharing') {
        await this.subscribeViewer();
      }
    } catch {
      await this.router.navigateByUrl('/r/invalid');
    }
  }

  private listen() {
    this.sub = this.events.connect(this.id(), this.sessionId).subscribe((event) => {
      void this.onEvent(event);
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
      const next = event.payload.state;
      this.state.set(next);
      this.publication = event.payload.publication ?? this.publication;
      if ((next === 'waiting' || next === 'failed') && this.role() === 'viewer') {
        this.playback.set(null);
      }
      if ((next === 'waiting' || next === 'failed') && this.role() === 'viewer') {
        if (next === 'waiting') this.publication = undefined;
        await this.media.stop();
      }
      if (next === 'sharing' && this.role() === 'viewer') {
        await this.subscribeViewer();
      }
      return;
    }
    if (event.type === 'publication.state') {
      this.publication = {
        id: event.payload.publicationId,
        transport: event.payload.transport,
        state: event.payload.state,
      };
      if (event.payload.state === 'ended' || event.payload.state === 'failed') {
        this.playback.set(null);
        if (this.role() === 'viewer') await this.media.stop();
      } else if (event.payload.state === 'live' && this.role() === 'viewer') {
        await this.subscribeViewer();
      }
      return;
    }
    if (event.type === 'presence') {
      this.applyParticipantCount(event.payload.participantCount);
      return;
    }
    if (event.type === 'media.state' && event.payload.role === this.role()) {
      this.onMediaState(event.payload.state);
    }
  }

  private async subscribeViewer() {
    try {
      const transport = this.publication?.transport;
      if (!transport) {
        return;
      }
      if (!this.mediaTransports().includes(transport)) {
        throw new Error(`Esta transmissão usa ${transport}, indisponível neste navegador.`);
      }
      await this.media.subscribe(
        this.id(),
        this.sessionId,
        (remote) => this.playback.set(toPlayback(remote)),
        (state) => this.onMediaState(state),
        transport,
      );
    } catch (error) {
      this.playback.set(null);
      this.mediaError.set(error instanceof Error ? error.message : 'Falha ao receber a transmissão.');
      this.state.set('failed');
    }
  }

  private onMediaState(state: MediaConnectionState) {
    if (state === 'connecting' && this.state() !== 'sharing') {
      this.state.set('connecting');
    } else if (state === 'connected') {
      this.state.set('sharing');
    } else if (state === 'reconnecting') {
      this.state.set('reconnecting');
    } else if (state === 'failed') {
      this.playback.set(null);
      if (!this.mediaError()) {
        const transport = this.publication?.transport ?? this.selectedTransport();
        this.mediaError.set(
          transport === 'websocket'
            ? 'A transmissão WebSocket foi interrompida.'
            : 'O WebRTC/UDP não alcançou o servidor. Atrás do Cloudflare use WebSocket / HTTPS.',
        );
      }
      this.state.set('failed');
    } else if (state === 'closed') {
      this.playback.set(null);
    }
  }
}

function toPlayback(remote: MediaStream | MediaPlayback): MediaPlayback {
  if (remote && typeof remote === 'object' && 'kind' in remote) {
    return remote;
  }
  return { kind: 'stream', stream: remote };
}
