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
        if (typeof el.play === 'function') {
          try {
            void el.play()?.catch(() => undefined);
          } catch {
            // jsdom does not implement HTMLMediaElement.play.
          }
        }
        this.seekLiveEdge(el);
      }
    });
  }

  seekLiveEdge(video: HTMLVideoElement) {
    if (this.playback()?.kind !== 'url') return;
    const ranges = video.buffered;
    if (ranges.length) {
      const start = ranges.start(0);
      const end = ranges.end(ranges.length - 1);
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return;
      if (video.currentTime < start + 0.05 || end - video.currentTime > 1.5) {
        video.currentTime = Math.max(start, end - 0.25);
      }
      return;
    }
    if (Number.isFinite(video.duration) && video.duration > 1) {
      video.currentTime = Math.max(0, video.duration - 0.25);
    }
  }
}
