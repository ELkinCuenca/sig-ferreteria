import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize, forkJoin } from 'rxjs';

import {
  ApiErrorResponse,
  CreateUserPayload,
  UpdateUserPayload,
  UserAdmin,
  UserRole,
} from '../../core/models/user-admin.models';
import { UserAdminApiService } from '../../core/services/user-admin-api.service';

type EditorMode = 'create' | 'edit';

@Component({
  selector: 'app-user-management',

  standalone: true,

  imports: [CommonModule, ReactiveFormsModule],

  templateUrl: './user-management.component.html',

  styleUrl: './user-management.component.scss',
})
export class UserManagementComponent implements OnInit {
  private readonly api = inject(UserAdminApiService);

  private readonly formBuilder = inject(FormBuilder);

  readonly users = signal<UserAdmin[]>([]);

  readonly roles = signal<UserRole[]>([]);

  readonly loading = signal(true);

  readonly saving = signal(false);

  readonly actionUserId = signal<number | null>(null);

  readonly editorMode = signal<EditorMode>('create');

  readonly selectedUser = signal<UserAdmin | null>(null);

  readonly passwordUser = signal<UserAdmin | null>(null);

  readonly errorMessage = signal('');

  readonly successMessage = signal('');

  readonly activeUsers = computed(
    () => this.users().filter((user) => user.estado === 'ACTIVO').length,
  );

  readonly blockedUsers = computed(
    () => this.users().filter((user) => user.estado === 'BLOQUEADO').length,
  );

  readonly administratorUsers = computed(
    () => this.users().filter((user) => user.rol === 'ADMINISTRADOR').length,
  );

  readonly createForm = this.formBuilder.nonNullable.group({
    id_rol: [0, [Validators.required, Validators.min(1)]],

    nombre_usuario: ['', [Validators.required, Validators.pattern(/^[a-z0-9][a-z0-9._-]{2,49}$/)]],

    nombres: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(100)]],

    apellidos: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(100)]],

    correo: ['', [Validators.required, Validators.email, Validators.maxLength(150)]],

    contrasena: [
      '',
      [
        Validators.required,
        Validators.minLength(12),
        Validators.maxLength(72),
        Validators.pattern(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^A-Za-z0-9]).+$/),
      ],
    ],

    confirmar_contrasena: ['', [Validators.required]],
  });

  readonly editForm = this.formBuilder.nonNullable.group({
    id_rol: [0, [Validators.required, Validators.min(1)]],

    nombres: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(100)]],

    apellidos: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(100)]],

    correo: ['', [Validators.required, Validators.email, Validators.maxLength(150)]],
  });

  readonly passwordForm = this.formBuilder.nonNullable.group({
    contrasena: [
      '',
      [
        Validators.required,
        Validators.minLength(12),
        Validators.maxLength(72),
        Validators.pattern(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^A-Za-z0-9]).+$/),
      ],
    ],

    confirmar_contrasena: ['', [Validators.required]],
  });

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.loading.set(true);
    this.clearMessages();

    forkJoin({
      users: this.api.getUsers(),

      roles: this.api.getRoles(),
    })
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: (response) => {
          this.users.set(response.users.usuarios);

          this.roles.set(response.roles.roles);

          if (this.editorMode() === 'create' && this.createForm.controls.id_rol.value === 0) {
            this.createForm.controls.id_rol.setValue(this.roles()[0]?.id_rol ?? 0);
          }
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error, 'No fue posible consultar los usuarios.'));
        },
      });
  }

  startCreate(): void {
    this.clearMessages();

    this.editorMode.set('create');
    this.selectedUser.set(null);
    this.passwordUser.set(null);

    this.createForm.reset({
      id_rol: this.roles()[0]?.id_rol ?? 0,

      nombre_usuario: '',
      nombres: '',
      apellidos: '',
      correo: '',
      contrasena: '',
      confirmar_contrasena: '',
    });
  }

  startEdit(user: UserAdmin): void {
    this.clearMessages();

    this.editorMode.set('edit');
    this.selectedUser.set(user);
    this.passwordUser.set(null);

    this.editForm.reset({
      id_rol: user.id_rol,
      nombres: user.nombres,
      apellidos: user.apellidos,
      correo: user.correo,
    });
  }

  submitCreate(): void {
    this.clearMessages();

    if (this.createForm.invalid) {
      this.createForm.markAllAsTouched();

      this.errorMessage.set('Revisa los campos obligatorios del nuevo usuario.');

      return;
    }

    const formValue = this.createForm.getRawValue();

    if (formValue.contrasena !== formValue.confirmar_contrasena) {
      this.errorMessage.set('Las contraseñas no coinciden.');

      return;
    }

    const payload: CreateUserPayload = {
      id_rol: Number(formValue.id_rol),

      nombre_usuario: formValue.nombre_usuario.trim().toLowerCase(),

      nombres: formValue.nombres.trim(),

      apellidos: formValue.apellidos.trim(),

      correo: formValue.correo.trim().toLowerCase(),

      contrasena: formValue.contrasena,
    };

    this.saving.set(true);

    this.api
      .createUser(payload)
      .pipe(
        finalize(() => {
          this.saving.set(false);
        }),
      )
      .subscribe({
        next: (user) => {
          this.users.update((currentUsers) => [...currentUsers, user]);

          this.successMessage.set(`El usuario ${user.nombre_usuario} fue creado correctamente.`);

          this.startCreate();
          this.successMessage.set(`El usuario ${user.nombre_usuario} fue creado correctamente.`);
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error, 'No fue posible crear el usuario.'));
        },
      });
  }

  submitEdit(): void {
    this.clearMessages();

    const selected = this.selectedUser();

    if (!selected) {
      this.errorMessage.set('No existe un usuario seleccionado.');

      return;
    }

    if (this.editForm.invalid) {
      this.editForm.markAllAsTouched();

      this.errorMessage.set('Revisa los datos editables del usuario.');

      return;
    }

    const formValue = this.editForm.getRawValue();

    const payload: UpdateUserPayload = {
      id_rol: Number(formValue.id_rol),

      nombres: formValue.nombres.trim(),

      apellidos: formValue.apellidos.trim(),

      correo: formValue.correo.trim().toLowerCase(),
    };

    this.saving.set(true);

    this.api
      .updateUser(selected.id_usuario, payload)
      .pipe(
        finalize(() => {
          this.saving.set(false);
        }),
      )
      .subscribe({
        next: (updatedUser) => {
          this.replaceUser(updatedUser);

          this.selectedUser.set(updatedUser);

          this.successMessage.set(
            `Los datos de ${updatedUser.nombre_usuario} fueron actualizados.`,
          );
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error, 'No fue posible actualizar el usuario.'));
        },
      });
  }

  changeState(user: UserAdmin): void {
    this.clearMessages();

    const nextState = user.estado === 'ACTIVO' ? 'INACTIVO' : 'ACTIVO';

    const actionText = nextState === 'ACTIVO' ? 'activar' : 'desactivar';

    const confirmed = window.confirm(
      `¿Confirmas que deseas ${actionText} la cuenta ${user.nombre_usuario}?`,
    );

    if (!confirmed) {
      return;
    }

    this.actionUserId.set(user.id_usuario);

    this.api
      .updateState(user.id_usuario, nextState)
      .pipe(
        finalize(() => {
          this.actionUserId.set(null);
        }),
      )
      .subscribe({
        next: (updatedUser) => {
          this.replaceUser(updatedUser);

          this.refreshSelectedUser(updatedUser);

          this.successMessage.set(
            `La cuenta ${updatedUser.nombre_usuario} quedó ${updatedUser.estado.toLowerCase()}.`,
          );
        },

        error: (error: unknown) => {
          this.errorMessage.set(
            this.extractError(error, 'No fue posible cambiar el estado de la cuenta.'),
          );
        },
      });
  }

  unlockUser(user: UserAdmin): void {
    this.clearMessages();

    const confirmed = window.confirm(`¿Confirmas el desbloqueo de ${user.nombre_usuario}?`);

    if (!confirmed) {
      return;
    }

    this.actionUserId.set(user.id_usuario);

    this.api
      .unlockUser(user.id_usuario)
      .pipe(
        finalize(() => {
          this.actionUserId.set(null);
        }),
      )
      .subscribe({
        next: (updatedUser) => {
          this.replaceUser(updatedUser);

          this.refreshSelectedUser(updatedUser);

          this.successMessage.set(`La cuenta ${updatedUser.nombre_usuario} fue desbloqueada.`);
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error, 'No fue posible desbloquear la cuenta.'));
        },
      });
  }

  openPasswordReset(user: UserAdmin): void {
    this.clearMessages();

    this.passwordUser.set(user);

    this.passwordForm.reset({
      contrasena: '',
      confirmar_contrasena: '',
    });
  }

  closePasswordReset(): void {
    this.passwordUser.set(null);

    this.passwordForm.reset({
      contrasena: '',
      confirmar_contrasena: '',
    });
  }

  submitPasswordReset(): void {
    this.clearMessages();

    const user = this.passwordUser();

    if (!user) {
      this.errorMessage.set('No existe una cuenta seleccionada.');

      return;
    }

    if (this.passwordForm.invalid) {
      this.passwordForm.markAllAsTouched();

      this.errorMessage.set('La nueva contraseña no cumple los requisitos.');

      return;
    }

    const formValue = this.passwordForm.getRawValue();

    if (formValue.contrasena !== formValue.confirmar_contrasena) {
      this.errorMessage.set('Las contraseñas no coinciden.');

      return;
    }

    this.saving.set(true);

    this.api
      .resetPassword(user.id_usuario, formValue.contrasena)
      .pipe(
        finalize(() => {
          this.saving.set(false);
        }),
      )
      .subscribe({
        next: (updatedUser) => {
          this.replaceUser(updatedUser);

          this.refreshSelectedUser(updatedUser);

          this.closePasswordReset();

          this.successMessage.set(
            `La contraseña de ${updatedUser.nombre_usuario} fue restablecida.`,
          );
        },

        error: (error: unknown) => {
          this.errorMessage.set(
            this.extractError(error, 'No fue posible restablecer la contraseña.'),
          );
        },
      });
  }

  roleDescription(roleId: number): string {
    return this.roles().find((role) => role.id_rol === roleId)?.descripcion ?? '';
  }

  stateLabel(state: string): string {
    switch (state) {
      case 'ACTIVO':
        return 'Activo';

      case 'INACTIVO':
        return 'Inactivo';

      case 'BLOQUEADO':
        return 'Bloqueado';

      default:
        return state;
    }
  }

  formatDate(value?: string): string {
    if (!value) {
      return 'Sin registro';
    }

    const date = new Date(value);

    if (Number.isNaN(date.getTime())) {
      return value;
    }

    return new Intl.DateTimeFormat('es-EC', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(date);
  }

  private replaceUser(updatedUser: UserAdmin): void {
    this.users.update((currentUsers) =>
      currentUsers.map((user) => (user.id_usuario === updatedUser.id_usuario ? updatedUser : user)),
    );
  }

  private refreshSelectedUser(updatedUser: UserAdmin): void {
    if (this.selectedUser()?.id_usuario === updatedUser.id_usuario) {
      this.selectedUser.set(updatedUser);

      this.editForm.patchValue({
        id_rol: updatedUser.id_rol,

        nombres: updatedUser.nombres,

        apellidos: updatedUser.apellidos,

        correo: updatedUser.correo,
      });
    }
  }

  private clearMessages(): void {
    this.errorMessage.set('');
    this.successMessage.set('');
  }

  private extractError(error: unknown, fallbackMessage: string): string {
    if (error instanceof HttpErrorResponse) {
      const response = error.error as ApiErrorResponse | undefined;

      if (typeof response?.message === 'string' && response.message.trim() !== '') {
        return response.message;
      }
    }

    return fallbackMessage;
  }
}
