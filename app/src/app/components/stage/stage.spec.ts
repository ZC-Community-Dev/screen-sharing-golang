import { ComponentRef } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { Stage } from './stage';

describe('Stage', () => {
  async function render(state: 'waiting' | 'sharing', stream: MediaStream | null) {
    await TestBed.configureTestingModule({ imports: [Stage] }).compileComponents();
    const fixture = TestBed.createComponent(Stage);
    const ref = fixture.componentRef as ComponentRef<Stage>;
    ref.setInput('state', state);
    ref.setInput('stream', stream);
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

  it('shows connecting copy while sharing without a stream', async () => {
    const fixture = await render('sharing', null);
    const el = fixture.nativeElement as HTMLElement;
    expect(el.querySelector('[data-testid="connecting"]')?.textContent).toContain('Conectando');
    expect(el.querySelector('[data-testid="waiting"]')).toBeNull();
    expect(el.querySelector('[data-testid="shared-screen"]')).toBeNull();
  });

  it('letterboxes the shared screen when sharing', async () => {
    const stream = { getTracks: () => [] } as unknown as MediaStream;
    const fixture = await render('sharing', stream);
    const video = fixture.nativeElement.querySelector('[data-testid="shared-screen"]') as HTMLVideoElement;
    expect(video).toBeTruthy();
    expect(video.className).toContain('object-contain');
    expect(fixture.nativeElement.textContent).not.toContain('câmera');
  });
});
