import { Component, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { RouterOutlet } from '@angular/router';
import { Issue, ManifestService, ValidateResponse } from './manifest.service';

// The example that is in the text box when the page opens.
const EXAMPLE_MANIFEST = `name: Billing Service
owner: team payments
url: http://billing.example.com
healthEndpoint: health
version: 1.2`;

@Component({
  imports: [RouterOutlet],
  selector: 'app-root',
  styleUrl: './app.css',
  templateUrl: './app.html',
})
export class App {
  private manifestService = inject(ManifestService);

  protected readonly title = signal('Service Manifest Validator');

  // The fields we show on the page, in this order.
  protected readonly fields = ['name', 'owner', 'url', 'healthEndpoint', 'version', 'description'];

  // A signal holds a value. When the value changes, the page updates by itself.
  protected readonly manifestText = signal(EXAMPLE_MANIFEST);
  protected readonly result = signal<ValidateResponse | null>(null);
  protected readonly errorMessage = signal('');
  protected readonly loading = signal(false);

  // Runs when the user clicks the Check button.
  protected check(): void {
    this.loading.set(true);
    this.result.set(null);
    this.errorMessage.set('');

    this.manifestService.validate(this.manifestText()).subscribe({
      // The server answered. Save the answer so the page can show it.
      next: (response) => {
        this.result.set(response);
        this.loading.set(false);
      },
      // Something went wrong. Show a message the user can understand.
      error: (error: HttpErrorResponse) => {
        this.errorMessage.set(this.readError(error));
        this.loading.set(false);
      },
    });
  }

  // Returns the issues for one field, so the page can show them next to it.
  protected issuesFor(field: string): Issue[] {
    return this.result()?.issues.filter((issue) => issue.field === field) ?? [];
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
