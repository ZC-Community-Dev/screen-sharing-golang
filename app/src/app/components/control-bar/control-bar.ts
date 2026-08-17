import { Component, input, output } from '@angular/core';

@Component({
  selector: 'app-control-bar',
  templateUrl: './control-bar.html',
})
export class ControlBar {
  readonly participantCount = input(0);
  readonly publicUrl = input('');
  readonly canPresent = input(false);
  readonly sharing = input(false);
  readonly copy = output<void>();
  readonly startShare = output<void>();
  readonly stopShare = output<void>();
}
