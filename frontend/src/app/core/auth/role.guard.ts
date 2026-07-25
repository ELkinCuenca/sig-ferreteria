import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map } from 'rxjs';

import { RoleName } from '../models/auth.models';
import { AuthService } from './auth.service';
import { AuthSessionService } from './auth-session.service';

export const roleGuard: CanActivateFn = (route, state) => {
  const auth = inject(AuthService);

  const session = inject(AuthSessionService);

  const router = inject(Router);

  const allowedRoles = (route.data?.['roles'] ?? []) as RoleName[];

  return auth.ensureSession().pipe(
    map((authenticated) => {
      if (!authenticated) {
        return router.createUrlTree(['/login'], {
          queryParams: {
            returnUrl: state.url,
          },
        });
      }

      if (allowedRoles.length > 0 && !session.hasAnyRole(allowedRoles)) {
        return router.createUrlTree(['/sin-permiso']);
      }

      return true;
    }),
  );
};
