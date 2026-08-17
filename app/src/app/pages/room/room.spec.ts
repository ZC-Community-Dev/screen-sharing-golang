import { provideHttpClient } from '@angular/common/http';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';

import { LinksService } from '../../services/links.service';
import { PresenterTokenStore } from '../../services/presenter-token.store';
import { RoomEventsService } from '../../services/room-events.service';
import { Room } from './room';

describe('Room', () => {
  it('shows share control only with a presenter token and never mic/cam', async () => {
    await TestBed.configureTestingModule({
      imports: [Room],
      providers: [
        provideHttpClient(),
        provideRouter([]),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: { get: () => 'Abcdefgh12' } } } },
        { provide: PresenterTokenStore, useValue: { get: () => 'TOKEN', save: () => undefined } },
        {
          provide: LinksService,
          useValue: {
            get: () => of({ id: 'Abcdefgh12', state: 'waiting', participantCount: 1 }),
            claimPresenter: () =>
              of({ sessionId: 's1', role: 'presenter', id: 'Abcdefgh12', state: 'waiting' }),
            joinViewer: () => of({ sessionId: 'v1', role: 'viewer', id: 'Abcdefgh12', state: 'waiting' }),
            startShare: () => of({ id: 'Abcdefgh12', state: 'sharing', participantCount: 1 }),
            stopShare: () => of({ id: 'Abcdefgh12', state: 'waiting', participantCount: 1 }),
          },
        },
        {
          provide: RoomEventsService,
          useValue: { connect: () => of(), send: () => undefined, whenOpen: () => Promise.resolve() },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(Room);
    await fixture.whenStable();
    fixture.detectChanges();
    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Compartilhar tela');
    expect(text.toLowerCase()).not.toContain('microfone');
    expect(text.toLowerCase()).not.toContain('câmera');
  });
});
