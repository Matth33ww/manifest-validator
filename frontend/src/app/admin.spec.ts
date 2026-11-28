import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Admin } from './admin';

const API = 'http://localhost:8080';

// A registered service, used in many tests.
const billing = {
  id: '1',
  name: 'billing',
  owner: 'pay@example.com',
  url: 'https://billing.example.com',
  healthEndpoint: '/health',
};

// Tests for the admin page.
describe('Admin', () => {
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Admin],
      // A fake HTTP backend: no real Go server is needed in these tests.
      providers: [provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();

    http = TestBed.inject(HttpTestingController);
  });

  // Fail the test if a request was sent but not answered.
  afterEach(() => http.verify());

  // Opens the page and answers the first request with this list of services.
  async function openWith(services: unknown[]) {
    const fixture = TestBed.createComponent(Admin);
    await fixture.whenStable();
    http.expectOne(`${API}/services`).flush(services);
    await fixture.whenStable();
    const page = fixture.nativeElement as HTMLElement;
    return { fixture, page };
  }

  // Types text into a box, the same way a user does.
  function type(page: HTMLElement, selector: string, text: string) {
    const box = page.querySelector(selector) as HTMLInputElement;
    box.value = text;
    box.dispatchEvent(new Event('input'));
  }

  // Finds a button by its text and clicks it.
  function click(page: HTMLElement, text: string) {
    const button = Array.from(page.querySelectorAll('button')).find((b) =>
      b.textContent?.includes(text),
    );
    button!.click();
  }

  it('should show a message when no services are registered', async () => {
    const { page } = await openWith([]);
    expect(page.querySelector('.empty')?.textContent).toContain('No services are registered yet');
  });

  it('should show each registered service', async () => {
    const { page } = await openWith([billing]);
    expect(page.querySelector('.service h2')?.textContent).toContain('billing');
    expect(page.querySelector('.owner')?.textContent).toContain('pay@example.com');
  });

  it('should show a clear message when the server is down', async () => {
    const fixture = TestBed.createComponent(Admin);
    await fixture.whenStable();
    // Status 0 means the browser could not reach the server at all.
    http.expectOne(`${API}/services`).error(new ProgressEvent('error'), { status: 0 });
    await fixture.whenStable();

    const page = fixture.nativeElement as HTMLElement;
    expect(page.querySelector('.error-box')?.textContent).toContain('Could not reach the server');
  });

  it('should add a new service to the list and clear the form', async () => {
    const { fixture, page } = await openWith([]);
    type(page, '#new-manifest', 'name: billing');
    await fixture.whenStable();
    click(page, 'Register service');

    const request = http.expectOne(`${API}/services`);
    expect(request.request.method).toBe('POST');
    request.flush(billing);
    await fixture.whenStable();

    expect(page.querySelector('.service h2')?.textContent).toContain('billing');
    expect((page.querySelector('#new-manifest') as HTMLTextAreaElement).value).toBe('');
  });

  it('should show the problems when the manifest is not valid', async () => {
    const { fixture, page } = await openWith([]);
    type(page, '#new-manifest', 'name: Bad Name');
    await fixture.whenStable();
    click(page, 'Register service');

    // 422 means the manifest has problems. The server sends them as a list.
    http.expectOne(`${API}/services`).flush(
      { valid: false, issues: [{ field: 'owner', message: 'owner is required', severity: 'error' }] },
      { status: 422, statusText: 'Unprocessable Entity' },
    );
    await fixture.whenStable();

    expect(page.querySelector('.register .error-box')?.textContent).toContain('has problems');
    expect(page.querySelector('.issues')?.textContent).toContain('owner: owner is required');
  });

  it('should put a new announcement on top', async () => {
    const { fixture, page } = await openWith([billing]);
    click(page, 'Announcements');

    // The page loads the announcements that are already there.
    const old = { id: '2', serviceId: '1', title: 'Old news', body: '', createdAt: '2026-11-01T10:00:00Z' };
    http.expectOne(`${API}/services/1/announcements`).flush([old]);
    await fixture.whenStable();

    type(page, '#new-title', 'New version');
    await fixture.whenStable();
    click(page, 'Post announcement');

    const request = http.expectOne(`${API}/services/1/announcements`);
    expect(request.request.body).toEqual({ title: 'New version', body: '' });
    request.flush({ id: '3', serviceId: '1', title: 'New version', body: '', createdAt: '2026-11-28T10:00:00Z' });
    await fixture.whenStable();

    const titles = Array.from(page.querySelectorAll('.announcement h3')).map((h) => h.textContent);
    expect(titles).toEqual(['New version', 'Old news']);
  });

  it('should not show old announcements after a quick click on another service', async () => {
    const search = { ...billing, id: '2', name: 'search', owner: 'find@example.com' };
    const { fixture, page } = await openWith([billing, search]);
    const buttons = page.querySelectorAll('button.small');

    // Click billing, then search, before the server answers.
    (buttons[0] as HTMLButtonElement).click();
    (buttons[1] as HTMLButtonElement).click();
    const forBilling = http.expectOne(`${API}/services/1/announcements`);
    const forSearch = http.expectOne(`${API}/services/2/announcements`);

    // The answer for search comes first, and the answer for billing comes last.
    forSearch.flush([{ id: '3', serviceId: '2', title: 'Search news', body: '', createdAt: '2026-11-28T10:00:00Z' }]);
    forBilling.flush([{ id: '4', serviceId: '1', title: 'Billing news', body: '', createdAt: '2026-11-28T10:00:00Z' }]);
    await fixture.whenStable();

    expect(page.querySelector('.announcements h2')?.textContent).toContain('search');
    const titles = Array.from(page.querySelectorAll('.announcement h3')).map((h) => h.textContent);
    expect(titles).toEqual(['Search news']);
  });
});
