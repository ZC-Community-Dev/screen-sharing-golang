import { Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { publicOrigin } from '../../config';
import { CreateLinkResponse, LinksService } from '../../services/links.service';
import { PresenterTokenStore } from '../../services/presenter-token.store';

@Component({
  selector: 'app-home',
  templateUrl: './home.html',
})
export class Home {
  private readonly links = inject(LinksService);
  private readonly tokens = inject(PresenterTokenStore);
  private readonly router = inject(Router);

  readonly busy = signal(false);
  readonly error = signal('');
  readonly created = signal<CreateLinkResponse | null>(null);

  async generate() {
    this.busy.set(true);
    this.error.set('');
    try {
      const created = await firstValueFrom(this.links.create());
      this.tokens.save(created.id, created.presenterToken);
      this.created.set(created);
      const url = `${publicOrigin()}${created.publicUrl}`;
      await navigator.clipboard.writeText(url);
    } catch {
      this.error.set('Não foi possível gerar o link.');
    } finally {
      this.busy.set(false);
    }
  }

  async enterRoom() {
    const created = this.created();
    if (!created) {
      return;
    }
    await this.router.navigateByUrl(created.publicUrl);
  }
}
