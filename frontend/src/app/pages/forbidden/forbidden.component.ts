import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-forbidden',
  standalone: true,
  imports: [RouterLink],
  template: `
    <section class="forbidden-page">
      <div class="forbidden-card">
        <span>403</span>

        <h1>Acceso no autorizado</h1>

        <p>Su rol no tiene permiso para acceder a este módulo.</p>

        <a routerLink="/"> Regresar al panel </a>
      </div>
    </section>
  `,
  styles: `
    .forbidden-page {
      min-height: 60vh;
      display: grid;
      place-items: center;
      padding: 1rem;
    }

    .forbidden-card {
      width: min(100%, 500px);
      padding: 2rem;
      border: 1px solid var(--border);
      border-radius: 1rem;
      background: var(--surface);
      box-shadow: var(--shadow);
      text-align: center;
    }

    .forbidden-card > span {
      color: var(--primary);
      font-size: 3.2rem;
      font-weight: 900;
    }

    .forbidden-card h1 {
      margin: 0.5rem 0;
    }

    .forbidden-card p {
      color: var(--muted);
    }

    .forbidden-card a {
      display: inline-block;
      margin-top: 0.7rem;
      padding: 0.7rem 1rem;
      border-radius: 0.7rem;
      background: var(--primary);
      color: white;
      font-weight: 800;
      text-decoration: none;
    }
  `,
})
export class ForbiddenComponent {}
