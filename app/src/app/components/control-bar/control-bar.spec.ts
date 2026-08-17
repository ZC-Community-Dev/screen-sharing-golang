import { ComponentRef } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { ControlBar } from './control-bar';

describe('ControlBar', () => {
  it('shows copy, people count, and no mic/cam/chat', async () => {
    await TestBed.configureTestingModule({ imports: [ControlBar] }).compileComponents();
    const fixture = TestBed.createComponent(ControlBar);
    const ref = fixture.componentRef as ComponentRef<ControlBar>;
    ref.setInput('participantCount', 3);
    ref.setInput('canPresent', true);
    ref.setInput('sharing', false);
    fixture.detectChanges();
    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Copiar link');
    expect(text).toContain('3 pessoas');
    expect(text).toContain('Compartilhar tela');
    expect(text.toLowerCase()).not.toContain('microfone');
    expect(text.toLowerCase()).not.toContain('câmera');
    expect(text.toLowerCase()).not.toContain('chat');
  });
});
