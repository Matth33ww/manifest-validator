import { Routes } from '@angular/router';
import { Admin } from './admin';
import { Validator } from './validator';

// Which page to show for each address.
export const routes: Routes = [
  // The start page is the validator.
  { path: '', component: Validator, title: 'Manifest Validator' },
  { path: 'admin', component: Admin, title: 'Admin - Manifest Validator' },
  // Any other address goes back to the start page.
  { path: '**', redirectTo: '' },
];
