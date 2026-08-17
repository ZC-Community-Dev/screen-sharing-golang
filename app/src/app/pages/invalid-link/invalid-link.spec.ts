import { provideRouter } from '@angular/router';
import { TestBed } from '@angular/core/testing';

import { InvalidLink } from './invalid-link';

describe('InvalidLink', () => {
  it('explains that the link is invalid', async () => {
    await TestBed.configureTestingModule({
      imports: [InvalidLink],
      providers: [provideRouter([])],
    }).compileComponents();
    const fixture = TestBed.createComponent(InvalidLink);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Link inválido');
  });
});
