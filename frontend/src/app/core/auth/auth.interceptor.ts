import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';

import { AuthSessionService } from './auth-session.service';

export const authInterceptor: HttpInterceptorFn = (request, next) => {
  const session = inject(AuthSessionService);

  const router = inject(Router);

  const token = session.token();

  const isApiRequest = request.url.startsWith('/api/v1/');

  const isLoginRequest = request.url.endsWith('/api/v1/auth/login');

  let authenticatedRequest = request;

  if (token && isApiRequest && !isLoginRequest) {
    authenticatedRequest = request.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`,
      },
    });
  }

  return next(authenticatedRequest).pipe(
    catchError((error: HttpErrorResponse) => {
      if (error.status === 401 && !isLoginRequest) {
        session.clear();

        void router.navigate(['/login'], {
          queryParams: {
            returnUrl: router.url !== '/login' ? router.url : '/',
          },
        });
      }

      return throwError(() => error);
    }),
  );
};
