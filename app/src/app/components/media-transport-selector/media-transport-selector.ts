import { Component, input, output } from '@angular/core';

import { MediaTransport } from '../../services/media-transport';

@Component({
  selector: 'app-media-transport-selector',
  templateUrl: './media-transport-selector.html',
})
export class MediaTransportSelector {
  readonly transports = input.required<readonly MediaTransport[]>();
  readonly selected = input.required<MediaTransport>();
  readonly selectedChange = output<MediaTransport>();

  label(transport: MediaTransport): string {
    return transport === 'webrtc' ? 'WebRTC / UDP' : 'WebSocket / HTTPS';
  }
}
