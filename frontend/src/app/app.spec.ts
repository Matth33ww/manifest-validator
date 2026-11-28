import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { App } from './app';

// Tests for the frame around the pages.
// The validator page has its own tests in validator.spec.ts.
describe('App', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [App],
      // The menu links need the router. An empty list of routes is enough here.
      providers: [provideRouter([])],
    }).compileComponents();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(App);
    expect(fixture.componentInstance).toBeTruthy();
  });

  it('should show the app name', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const page = fixture.nativeElement as HTMLElement;
    expect(page.textContent).toContain('Service Manifest Validator');
  });

  it('should have menu links to both pages', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const page = fixture.nativeElement as HTMLElement;

    // Read the text and the address of each menu link.
    const links = Array.from(page.querySelectorAll('nav a')).map((link) => ({
      text: link.textContent?.trim(),
      href: link.getAttribute('href'),
    }));

    expect(links).toEqual([
      { text: 'Validator', href: '/' },
      { text: 'Admin', href: '/admin' },
    ]);
  });
});
