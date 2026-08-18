import { afterRenderEffect, Component, ElementRef, input, viewChild } from '@angular/core';

import { RoomState } from '../../services/links.service';
import { MediaPlayback } from '../../services/media-transport';

@Component({
  selector: 'app-stage',
  templateUrl: './stage.html',
})
export class Stage {
  readonly state = input<RoomState>('waiting');
  readonly playback = input<MediaPlayback | null>(null);
  private readonly video = viewChild<ElementRef<HTMLVideoElement>>('screen');

  constructor() {
    afterRenderEffect(() => {
      const el = this.video()?.nativeElement;
      if (el) {
        const playback = this.playback();
        if (playback?.kind === 'stream') {
          el.removeAttribute('src');
          el.srcObject = playback.stream;
        } else {
          el.srcObject = null;
          if (playback?.kind === 'url' && el.src !== playback.url) el.src = playback.url;
        }
      }
    });
  }

  seekLiveEdge(video: HTMLVideoElement) {
    if (this.playback()?.kind === 'url' && Number.isFinite(video.duration) && video.duration > 1) {
      video.currentTime = Math.max(0, video.duration - 0.25);
    }
  }
}
