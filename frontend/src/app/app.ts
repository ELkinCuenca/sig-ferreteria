import { Component, inject, signal } from '@angular/core';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { finalize } from 'rxjs';

import { AuthService } from './core/auth/auth.service';
import { AuthSessionService } from './core/auth/auth-session.service';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  templateUrl: './app.html',
  styleUrl: './app.scss',
})
export class App {
  private readonly auth = inject(AuthService);

  private readonly router = inject(Router);

  readonly session = inject(AuthSessionService);

  readonly currentYear = new Date().getFullYear();

  readonly loggingOut = signal(false);

  logout(): void {
    if (this.loggingOut()) {
      return;
    }

    this.loggingOut.set(true);

    this.auth
      .logout()
      .pipe(
        finalize(() => {
          this.loggingOut.set(false);

          void this.router.navigate(['/login']);
        }),
      )
      .subscribe({
        error: (error: unknown) => {
          console.error('No se pudo cerrar la sesión remota:', error);
        },
      });
  }
}
