import { provideHttpClient } from '@angular/common/http';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { TestBed } from '@angular/core/testing';
import { Subject, of } from 'rxjs';
import { vi } from 'vitest';

import { LinksService } from '../../services/links.service';
import { MediaService } from '../../services/media.service';
import { PresenterTokenStore } from '../../services/presenter-token.store';
import { RoomEvent } from '../../services/room-events.service';
import { RoomEventsService } from '../../services/room-events.service';
import { Room } from './room';

describe('Room', () => {
  const media = {
    loadTransports: vi.fn().mockResolvedValue(['webrtc']),
    defaultTransport: 'webrtc',
    publish: vi.fn(),
    subscribe: vi.fn(),
    stop: vi.fn().mockResolvedValue(undefined),
  };

  async function render(token: string | null, initialState: 'waiting' | 'sharing' = 'waiting') {
    const roomEvents = new Subject<RoomEvent>();
    const links = {
      get: () =>
        of({
          id: 'Abcdefgh12',
          state: initialState,
          participantCount: 1,
          publication:
            initialState === 'sharing'
              ? { id: 'pub-1', transport: 'webrtc', state: 'live' }
              : null,
        }),
      claimPresenter: () =>
        of({ sessionId: 's1', role: 'presenter', id: 'Abcdefgh12', state: initialState }),
      joinViewer: () =>
        of({ sessionId: 'v1', role: 'viewer', id: 'Abcdefgh12', state: initialState }),
      stopShare: () => of({ id: 'Abcdefgh12', state: 'waiting', participantCount: 1 }),
    };
    await TestBed.configureTestingModule({
      imports: [Room],
      providers: [
        provideHttpClient(),
        provideRouter([]),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: { get: () => 'Abcdefgh12' } } } },
        { provide: PresenterTokenStore, useValue: { get: () => token } },
        { provide: LinksService, useValue: links },
        { provide: MediaService, useValue: media },
        {
          provide: RoomEventsService,
          useValue: { connect: () => roomEvents.asObservable() },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(Room);
    await fixture.whenStable();
    fixture.detectChanges();
    return { fixture, roomEvents };
  }

  beforeEach(() => {
    vi.clearAllMocks();
    media.publish.mockReset();
    media.subscribe.mockReset();
    media.stop.mockResolvedValue(undefined);
    media.loadTransports.mockResolvedValue(['webrtc']);
    media.defaultTransport = 'webrtc';
  });

  it('shows share control only with a presenter token and never mic/cam', async () => {
    const { fixture } = await render('TOKEN');
    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Compartilhar tela');
    expect(text.toLowerCase()).not.toContain('microfone');
    expect(text.toLowerCase()).not.toContain('câmera');
  });

  it('enters connecting and creates only one publisher session', async () => {
    const display = {} as MediaStream;
    media.publish.mockResolvedValue(display);
    const { fixture } = await render('TOKEN');

    await fixture.componentInstance.startShare();

    expect(media.publish).toHaveBeenCalledOnce();
    expect(media.publish).toHaveBeenCalledWith(
      'Abcdefgh12',
      's1',
      'TOKEN',
      expect.any(Function),
      'webrtc',
    );
    expect(fixture.componentInstance.state()).toBe('connecting');
  });

  it('subscribes on sharing and tears down on waiting without signaling participants', async () => {
    media.subscribe.mockResolvedValue(undefined);
    const { fixture, roomEvents } = await render(null);

    roomEvents.next({
      type: 'room.state',
      payload: {
        state: 'sharing',
        publication: { id: 'pub-1', transport: 'webrtc', state: 'live' },
      },
    });
    await fixture.whenStable();
    expect(media.subscribe).toHaveBeenCalledWith(
      'Abcdefgh12',
      'v1',
      expect.any(Function),
      expect.any(Function),
      'webrtc',
    );

    roomEvents.next({ type: 'room.state', payload: { state: 'waiting' } });
    await fixture.whenStable();
    expect(media.stop).toHaveBeenCalled();
    expect(fixture.componentInstance.playback()).toBeNull();
  });

  it('subscribes immediately when a viewer joins an active share', async () => {
    media.subscribe.mockResolvedValue(undefined);

    await render(null, 'sharing');

    expect(media.subscribe).toHaveBeenCalledOnce();
    expect(media.publish).not.toHaveBeenCalled();
  });

  it('shows transport choice only to a presenter before capture', async () => {
    media.loadTransports.mockResolvedValue(['webrtc', 'websocket']);
    const presenter = await render('TOKEN');
    expect(
      presenter.fixture.nativeElement.querySelector('app-media-transport-selector'),
    ).toBeTruthy();

    TestBed.resetTestingModule();
    const viewer = await render(null);
    expect(viewer.fixture.nativeElement.querySelector('app-media-transport-selector')).toBeNull();
  });

  it('does not subscribe until the publication transport is known', async () => {
    media.subscribe.mockResolvedValue(undefined);
    const { fixture, roomEvents } = await render(null);

    roomEvents.next({ type: 'room.state', payload: { state: 'sharing' } });
    await fixture.whenStable();
    expect(media.subscribe).not.toHaveBeenCalled();

    roomEvents.next({
      type: 'publication.state',
      payload: { publicationId: 'pub-1', transport: 'webrtc', state: 'live' },
    });
    await fixture.whenStable();
    expect(media.subscribe).toHaveBeenCalledWith(
      'Abcdefgh12',
      'v1',
      expect.any(Function),
      expect.any(Function),
      'webrtc',
    );
  });

  it('subscribes with the publication transport instead of defaulting to webrtc', async () => {
    media.loadTransports.mockResolvedValue(['webrtc', 'websocket']);
    media.subscribe.mockResolvedValue(undefined);
    const { fixture, roomEvents } = await render(null);

    roomEvents.next({
      type: 'room.state',
      payload: {
        state: 'sharing',
        publication: { id: 'pub-1', transport: 'websocket', state: 'live' },
      },
    });
    await fixture.whenStable();
    expect(media.subscribe).toHaveBeenCalledWith(
      'Abcdefgh12',
      'v1',
      expect.any(Function),
      expect.any(Function),
      'websocket',
    );
  });

  it('marks presenter media as sharing when connected', async () => {
    const display = {} as MediaStream;
    media.publish.mockImplementation(async (_id, _session, _token, onState) => {
      onState?.('connected');
      return display;
    });
    const { fixture } = await render('TOKEN');
    await fixture.componentInstance.startShare();
    expect(fixture.componentInstance.state()).toBe('sharing');
    expect(fixture.componentInstance.playback()).toEqual({ kind: 'stream', stream: display });
  });

  it('keeps playback while reconnecting', async () => {
    media.subscribe.mockResolvedValue(undefined);
    const { fixture, roomEvents } = await render(null);
    roomEvents.next({
      type: 'room.state',
      payload: {
        state: 'sharing',
        publication: { id: 'pub-1', transport: 'webrtc', state: 'live' },
      },
    });
    await fixture.whenStable();
    expect(media.subscribe).toHaveBeenCalled();
    const playback = { kind: 'stream' as const, stream: { getTracks: () => [] } as unknown as MediaStream };
    const onRemote = media.subscribe.mock.calls[0][2] as (remote: typeof playback) => void;
    const onState = media.subscribe.mock.calls[0][3] as (state: 'reconnecting') => void;
    onRemote(playback);
    onState('reconnecting');
    expect(fixture.componentInstance.playback()).toEqual(playback);
    expect(fixture.componentInstance.state()).toBe('reconnecting');
  });
});
