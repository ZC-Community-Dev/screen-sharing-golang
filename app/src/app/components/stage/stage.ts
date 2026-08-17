import { afterRenderEffect, Component, ElementRef, input, viewChild } from '@angular/core';

@Component({
  selector: 'app-stage',
  templateUrl: './stage.html',
})
export class Stage {
  readonly state = input<'waiting' | 'sharing'>('waiting');
  readonly stream = input<MediaStream | null>(null);
  private readonly video = viewChild<ElementRef<HTMLVideoElement>>('screen');

  constructor() {
    afterRenderEffect(() => {
      const el = this.video()?.nativeElement;
      if (el) {
        el.srcObject = this.stream();
      }
    });
  }
}
