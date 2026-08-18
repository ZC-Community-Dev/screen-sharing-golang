import { ComponentRef } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { MediaTransportSelector } from './media-transport-selector';

describe('MediaTransportSelector', () => {
  it('shows only available transports and emits an explicit choice', async () => {
    await TestBed.configureTestingModule({ imports: [MediaTransportSelector] }).compileComponents();
    const fixture = TestBed.createComponent(MediaTransportSelector);
    const ref = fixture.componentRef as ComponentRef<MediaTransportSelector>;
    ref.setInput('transports', ['webrtc', 'websocket']);
    ref.setInput('selected', 'webrtc');
    const chosen: string[] = [];
    fixture.componentInstance.selectedChange.subscribe((value) => chosen.push(value));
    fixture.detectChanges();

    const websocket = fixture.nativeElement.querySelector(
      '[data-testid="transport-websocket"]',
    ) as HTMLInputElement;
    expect(websocket).toBeTruthy();
    websocket.click();
    expect(chosen).toEqual(['websocket']);
  });

  it('does not render an unsupported WebSocket option', async () => {
    await TestBed.configureTestingModule({ imports: [MediaTransportSelector] }).compileComponents();
    const fixture = TestBed.createComponent(MediaTransportSelector);
    fixture.componentRef.setInput('transports', ['webrtc']);
    fixture.componentRef.setInput('selected', 'webrtc');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('[data-testid="transport-websocket"]')).toBeNull();
  });
});
