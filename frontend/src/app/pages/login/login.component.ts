import { HttpErrorResponse } from '@angular/common/http';
import { Component, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { finalize } from 'rxjs';

import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [ReactiveFormsModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
})
export class LoginComponent implements OnInit {
  private readonly formBuilder = inject(FormBuilder);

  private readonly auth = inject(AuthService);

  private readonly router = inject(Router);

  private readonly route = inject(ActivatedRoute);

  readonly submitting = signal(false);
  readonly checkingSession = signal(true);
  readonly errorMessage = signal('');
  readonly showPassword = signal(false);

  readonly form = this.formBuilder.nonNullable.group({
    usuario: ['', [Validators.required, Validators.maxLength(150)]],

    contrasena: ['', [Validators.required, Validators.maxLength(72)]],
  });

  ngOnInit(): void {
    this.auth
      .ensureSession()
      .pipe(
        finalize(() => {
          this.checkingSession.set(false);
        }),
      )
      .subscribe({
        next: (authenticated) => {
          if (authenticated) {
            void this.navigateAfterLogin();
          }
        },
      });
  }

  submit(): void {
    if (this.form.invalid || this.submitting()) {
      this.form.markAllAsTouched();
      return;
    }

    this.submitting.set(true);
    this.errorMessage.set('');

    this.auth
      .login(this.form.getRawValue())
      .pipe(
        finalize(() => {
          this.submitting.set(false);
        }),
      )
      .subscribe({
        next: () => {
          this.form.controls.contrasena.setValue('');

          void this.navigateAfterLogin();
        },

        error: (error: unknown) => {
          this.form.controls.contrasena.setValue('');

          this.errorMessage.set(this.extractError(error));
        },
      });
  }

  togglePassword(): void {
    this.showPassword.update((visible) => !visible);
  }

  private navigateAfterLogin(): Promise<boolean> {
    const requestedUrl = this.route.snapshot.queryParamMap.get('returnUrl');

    const destination =
      requestedUrl && requestedUrl.startsWith('/') && !requestedUrl.startsWith('//')
        ? requestedUrl
        : '/';

    return this.router.navigateByUrl(destination);
  }

  private extractError(error: unknown): string {
    if (
      error instanceof HttpErrorResponse &&
      error.error &&
      typeof error.error.message === 'string'
    ) {
      return error.error.message;
    }

    return 'No fue posible iniciar sesión. ' + 'Verifique la conexión con el servidor.';
  }
}
