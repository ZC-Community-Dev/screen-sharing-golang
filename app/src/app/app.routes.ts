import { Routes } from '@angular/router';

import { Home } from './pages/home/home';
import { InvalidLink } from './pages/invalid-link/invalid-link';
import { Room } from './pages/room/room';

export const routes: Routes = [
  { path: '', component: Home },
  { path: 'r/invalid', component: InvalidLink },
  { path: 'r/:id', component: Room },
];
