import { ComponentRef } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { RoomState } from '../../services/links.service';
import { MediaPlayback } from '../../services/media-transport';
import { Stage } from './stage';

describe('Stage', () => {
  async function render(state: RoomState, playback: MediaPlayback | null) {
    await TestBed.configureTestingModule({ imports: [Stage] }).compileComponents();
    const fixture = TestBed.createComponent(Stage);
    const ref = fixture.componentRef as ComponentRef<Stage>;
    ref.setInput('state', state);
    ref.setInput('playback', playback);
    fixture.detectChanges();
    return fixture;
  }

  it('shows waiting copy without a camera tile', async () => {
    const fixture = await render('waiting', null);
    const el = fixture.nativeElement as HTMLElement;
    expect(el.querySelector('[data-testid="waiting"]')?.textContent).toContain('Aguardando');
    expect(el.textContent).not.toContain('câmera');
    expect(el.querySelector('[data-testid="shared-screen"]')).toBeNull();
  });

  it('shows connecting copy before media arrives', async () => {
    const fixture = await render('connecting', null);
    const el = fixture.nativeElement as HTMLElement;
    expect(el.querySelector('[data-testid="connecting"]')?.textContent).toContain('Conectando');
    expect(el.querySelector('[data-testid="waiting"]')).toBeNull();
    expect(el.querySelector('[data-testid="shared-screen"]')).toBeNull();
  });

  it.each([
    ['reconnecting', 'Reconectando'],
    ['failed', 'Não foi possível'],
  ] as const)('shows the %s media state without a stale video', async (state, copy) => {
    const fixture = await render(state, { kind: 'stream', stream: {} as MediaStream });
    const el = fixture.nativeElement as HTMLElement;
    expect(el.textContent).toContain(copy);
    expect(el.querySelector('[data-testid="shared-screen"]')).toBeNull();
  });

  it('letterboxes the shared screen when sharing', async () => {
    const stream = { getTracks: () => [] } as unknown as MediaStream;
    const fixture = await render('sharing', { kind: 'stream', stream });
    const video = fixture.nativeElement.querySelector('[data-testid="shared-screen"]') as HTMLVideoElement;
    expect(video).toBeTruthy();
    expect(video.className).toContain('object-contain');
    expect(fixture.nativeElement.textContent).not.toContain('câmera');
  });

  it('renders a MediaSource object URL without assigning it as srcObject', async () => {
    const fixture = await render('sharing', { kind: 'url', url: 'blob:screen-share' });
    const video = fixture.nativeElement.querySelector('[data-testid="shared-screen"]') as HTMLVideoElement;
    expect(video.getAttribute('src')).toBe('blob:screen-share');
    expect(video.srcObject).toBeNull();
  });
});
