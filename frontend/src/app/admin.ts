import { Component, OnInit, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Issue } from './manifest.service';
import { RegisteredService, ServicesService } from './services';

// The admin page. It shows the registered services and lets the admin add a new one.
@Component({
  selector: 'app-admin',
  templateUrl: './admin.html',
})
export class Admin implements OnInit {
  private servicesService = inject(ServicesService);

  // A signal holds a value. When the value changes, the page updates by itself.
  protected readonly services = signal<RegisteredService[]>([]);
  protected readonly errorMessage = signal('');
  protected readonly loading = signal(false);

  // The values for the register form.
  protected readonly manifestText = signal('');
  protected readonly registering = signal(false);
  protected readonly registerError = signal('');
  // The problems the server found in the manifest we tried to register.
  protected readonly registerIssues = signal<Issue[]>([]);

  // Angular calls this once, when the page opens.
  ngOnInit(): void {
    this.loadServices();
  }

  // Asks the server for the list of services.
  protected loadServices(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    this.servicesService.list().subscribe({
      // The server answered. Save the list so the page can show it.
      next: (list) => {
        this.services.set(list);
        this.loading.set(false);
      },
      // Something went wrong. Show a message the user can understand.
      error: (error: HttpErrorResponse) => {
        this.errorMessage.set(this.readError(error));
        this.loading.set(false);
      },
    });
  }

  // Runs when the admin clicks the Register button.
  protected register(): void {
    this.registering.set(true);
    this.registerError.set('');
    this.registerIssues.set([]);

    this.servicesService.register(this.manifestText()).subscribe({
      // The service was saved. Add it to the list and clear the form.
      next: (service) => {
        this.services.update((list) => [...list, service]);
        this.manifestText.set('');
        this.registering.set(false);
      },
      error: (error: HttpErrorResponse) => {
        if (error.status === 422) {
          // 422 means the manifest has problems. The server sends them as a list.
          this.registerError.set('This manifest has problems. Fix them and try again.');
          this.registerIssues.set(error.error?.issues ?? []);
        } else {
          this.registerError.set(this.readError(error));
        }
        this.registering.set(false);
      },
    });
  }

  // Turns a failed request into a short message.
  private readError(error: HttpErrorResponse): string {
    if (error.status === 0) {
      return 'Could not reach the server. Is it running on port 8080?';
    }
    // The server sends errors as {"error": "..."}.
    return error.error?.error ?? 'Something went wrong. Please try again.';
  }
}
