import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

// One problem found in a manifest.
export interface Issue {
  field: string;
  message: string;
  severity: 'error' | 'warning';
}

// The answer we get back from POST /validate.
export interface ValidateResponse {
  valid: boolean;
  issues: Issue[];
}

// The address of the Go server. Change it if you run the server on another port.
const API_URL = 'http://localhost:8080';

@Injectable({ providedIn: 'root' })
export class ManifestService {
  private http = inject(HttpClient);

  // Sends the manifest text (YAML or JSON) to the server.
  validate(manifestText: string): Observable<ValidateResponse> {
    return this.http.post<ValidateResponse>(`${API_URL}/validate`, manifestText, {
      headers: { 'Content-Type': 'text/plain' },
    });
  }
}
