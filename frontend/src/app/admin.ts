import { Component, OnInit, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Issue } from './manifest.service';
import { Announcement, RegisteredService, ServicesService } from './services';

// The admin page. The admin can see the registered services, add a new one,
// and write announcements for a service.
@Component({
  // DatePipe lets the page show a date in a form people can read.
  imports: [DatePipe],
  selector: 'app-admin',
  styleUrl: './admin.css',
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

  // The service the admin clicked on. It is null when no service is open.
  protected readonly selected = signal<RegisteredService | null>(null);
  protected readonly announcements = signal<Announcement[]>([]);
  protected readonly announcementError = signal('');

  // The values for the new announcement form.
  protected readonly newTitle = signal('');
  protected readonly newBody = signal('');
  protected readonly posting = signal(false);

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

  // Runs when the admin clicks on a service. It opens the service and loads its announcements.
  protected select(service: RegisteredService): void {
    this.selected.set(service);
    this.announcements.set([]);
    this.announcementError.set('');

    this.servicesService.announcements(service.id).subscribe({
      next: (list) => {
        // The admin may have clicked another service while we waited.
        // Then this answer is old, so we do not show it.
        if (this.selected()?.id !== service.id) {
          return;
        }
        this.announcements.set(list);
      },
      error: (error: HttpErrorResponse) => {
        if (this.selected()?.id !== service.id) {
          return;
        }
        this.announcementError.set(this.readError(error));
      },
    });
  }

  // Runs when the admin clicks the Post button.
  protected addAnnouncement(): void {
    const service = this.selected();
    if (!service) {
      return;
    }
    this.posting.set(true);
    this.announcementError.set('');

    this.servicesService.addAnnouncement(service.id, this.newTitle(), this.newBody()).subscribe({
      // The announcement was saved. Put it first, because the newest is on top.
      next: (announcement) => {
        this.announcements.update((list) => [announcement, ...list]);
        this.newTitle.set('');
        this.newBody.set('');
        this.posting.set(false);
      },
      error: (error: HttpErrorResponse) => {
        this.announcementError.set(this.readError(error));
        this.posting.set(false);
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
