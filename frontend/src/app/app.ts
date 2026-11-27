import { Component, signal } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';

// The frame around every page: the top bar with the menu.
// The pages themselves are in validator.ts and admin.ts.
@Component({
  // RouterLink makes the menu links work. RouterLinkActive marks the open page.
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  selector: 'app-root',
  styleUrl: './app.css',
  templateUrl: './app.html',
})
export class App {
  protected readonly title = signal('Service Manifest Validator');
}
