import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { App } from './app';

describe('App', () => {
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [App],
      // A fake HTTP backend: no real Go server is needed in these tests.
      providers: [provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();

    http = TestBed.inject(HttpTestingController);
  });

  // Fail the test if a request was sent but not answered.
  afterEach(() => http.verify());

  // Opens the page and clicks the Check button.
  async function openAndCheck() {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const page = fixture.nativeElement as HTMLElement;
    page.querySelector('button')!.click();
    const request = http.expectOne('http://localhost:8080/validate');
    return { fixture, page, request };
  }

  it('should create the app', () => {
    const fixture = TestBed.createComponent(App);
    expect(fixture.componentInstance).toBeTruthy();
  });

  it('should render title', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const page = fixture.nativeElement as HTMLElement;
    expect(page.querySelector('h1')?.textContent).toContain('Service Manifest Validator');
  });

  it('should send the manifest text to the server', async () => {
    const { request } = await openAndCheck();
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toContain('name: Billing Service');
    request.flush({ valid: true, issues: [] });
  });

  it('should show each problem next to its field', async () => {
    const { fixture, page, request } = await openAndCheck();
    request.flush({
      valid: false,
      issues: [{ field: 'owner', message: 'owner must be a valid email address', severity: 'error' }],
    });
    await fixture.whenStable();

    expect(page.querySelector('.status')?.textContent).toContain('has problems');
    const ownerRow = Array.from(page.querySelectorAll('.field')).find((row) =>
      row.textContent?.includes('owner'),
    );
    expect(ownerRow?.textContent).toContain('owner must be a valid email address');
  });

  it('should say valid when there are no problems', async () => {
    const { fixture, page, request } = await openAndCheck();
    request.flush({ valid: true, issues: [] });
    await fixture.whenStable();

    expect(page.querySelector('.status')?.textContent).toContain('valid');
    expect(page.querySelectorAll('.issue').length).toBe(0);
  });

  it('should show a clear message when the server is down', async () => {
    const { fixture, page, request } = await openAndCheck();
    // Status 0 means the browser could not reach the server at all.
    request.error(new ProgressEvent('error'), { status: 0 });
    await fixture.whenStable();

    expect(page.querySelector('.error-box')?.textContent).toContain('Could not reach the server');
  });

  it('should show the error message from the server', async () => {
    const { fixture, page, request } = await openAndCheck();
    request.flush({ error: 'body is empty' }, { status: 400, statusText: 'Bad Request' });
    await fixture.whenStable();

    expect(page.querySelector('.error-box')?.textContent).toContain('body is empty');
  });
});
