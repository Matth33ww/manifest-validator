import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { API_URL } from './api';

// A service that has been registered after its manifest was valid.
export interface RegisteredService {
  id: string;
  name: string;
  owner: string;
  url: string;
  healthEndpoint: string;
  version?: string;
  description?: string;
}

// A news item or planned change for one registered service.
export interface Announcement {
  id: string;
  serviceId: string;
  title: string;
  body: string;
  createdAt: string;
}

@Injectable({ providedIn: 'root' })
export class ServicesService {
  private http = inject(HttpClient);

  // Gets every registered service.
  list(): Observable<RegisteredService[]> {
    return this.http.get<RegisteredService[]>(`${API_URL}/services`);
  }

  // Sends a manifest. The server checks it again, and saves it only if it is valid.
  register(manifestText: string): Observable<RegisteredService> {
    return this.http.post<RegisteredService>(`${API_URL}/services`, manifestText, {
      headers: { 'Content-Type': 'text/plain' },
    });
  }

  // Gets the announcements for one service.
  announcements(serviceId: string): Observable<Announcement[]> {
    return this.http.get<Announcement[]>(`${API_URL}/services/${serviceId}/announcements`);
  }

  // Adds a new announcement to a service.
  addAnnouncement(serviceId: string, title: string, body: string): Observable<Announcement> {
    return this.http.post<Announcement>(`${API_URL}/services/${serviceId}/announcements`, { title, body });
  }
}
