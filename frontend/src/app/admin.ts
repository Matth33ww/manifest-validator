import { Component, OnInit, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { RegisteredService, ServicesService } from './services';

// The admin page. It shows the services that are registered.
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

  // Turns a failed request into a short message.
  private readError(error: HttpErrorResponse): string {
    if (error.status === 0) {
      return 'Could not reach the server. Is it running on port 8080?';
    }
    // The server sends errors as {"error": "..."}.
    return error.error?.error ?? 'Something went wrong. Please try again.';
  }
}
