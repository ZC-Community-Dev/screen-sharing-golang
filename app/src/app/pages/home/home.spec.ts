import { provideHttpClient } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';

import { LinksService } from '../../services/links.service';
import { Home } from './home';

describe('Home', () => {
  it('copies the public URL only when generating a link', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    await TestBed.configureTestingModule({
      imports: [Home],
      providers: [
        provideHttpClient(),
        provideRouter([]),
        {
          provide: LinksService,
          useValue: {
            create: () =>
              of({
                id: 'Abcdefgh12',
                publicUrl: '/r/Abcdefgh12',
                presenterToken: 'SECRETTOKEN',
              }),
          },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(Home);
    fixture.detectChanges();
    const button = fixture.nativeElement.querySelector('button') as HTMLButtonElement;
    expect(button.textContent).toContain('Gerar link');
    button.click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(writeText).toHaveBeenCalled();
    const copied = writeText.mock.calls[0][0] as string;
    expect(copied).toContain('/r/Abcdefgh12');
    expect(copied).not.toContain('SECRETTOKEN');

    const token = fixture.nativeElement.querySelector(
      '[data-testid="presenter-token"]',
    ) as HTMLElement;
    expect(token?.textContent).toContain('SECRETTOKEN');
    const publicUrl = fixture.nativeElement.querySelector(
      '[data-testid="public-url"]',
    ) as HTMLElement;
    expect(publicUrl?.textContent).toContain('/r/Abcdefgh12');
    expect(publicUrl?.textContent).not.toContain('SECRETTOKEN');
  });
});

