import { HttpErrorResponse } from '@angular/common/http';
import { CommonModule } from '@angular/common';
import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize, forkJoin, Observable } from 'rxjs';

import { AuthSessionService } from '../../core/auth/auth-session.service';
import {
  BPMAction,
  BPMCreatePayload,
  BPMKanbanColumn,
  BPMProvider,
  BPMReceivePayload,
  BPMRejectPayload,
  BPMReplenishment,
  BPMReplenishmentDetail,
  BPMState,
  BPMTransitionPayload,
} from '../../core/models/bpm.models';
import { BPMApiService } from '../../core/services/bpm-api.service';

interface ActionDialog {
  action: BPMAction;
  item: BPMReplenishment;
}

@Component({
  selector: 'app-bpm',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './bpm.component.html',
  styleUrl: './bpm.component.scss',
})
export class BPMComponent implements OnInit {
  private readonly api = inject(BPMApiService);

  private readonly formBuilder = inject(FormBuilder);

  readonly session = inject(AuthSessionService);

  readonly providers = signal<BPMProvider[]>([]);

  readonly replenishments = signal<BPMReplenishment[]>([]);

  readonly selected = signal<BPMReplenishmentDetail | null>(null);

  readonly actionDialog = signal<ActionDialog | null>(null);

  readonly showCreate = signal(false);

  readonly loadingBoard = signal(false);

  readonly loadingDetail = signal(false);

  readonly saving = signal(false);

  readonly errorMessage = signal('');

  readonly successMessage = signal('');

  readonly columns: readonly BPMKanbanColumn[] = [
    {
      state: 'BORRADOR',
      title: 'Borradores',
      description: 'Pendientes de envío',
    },
    {
      state: 'SOLICITADA',
      title: 'Solicitadas',
      description: 'Esperan decisión',
    },
    {
      state: 'APROBADA',
      title: 'Aprobadas',
      description: 'Pendientes de pedido',
    },
    {
      state: 'EN_PEDIDO',
      title: 'En pedido',
      description: 'Esperan recepción',
    },
    {
      state: 'RECIBIDA',
      title: 'Recibidas',
      description: 'Pendientes de cierre',
    },
    {
      state: 'CERRADA',
      title: 'Cerradas',
      description: 'Procesos finalizados',
    },
    {
      state: 'RECHAZADA',
      title: 'Rechazadas',
      description: 'Solicitudes denegadas',
    },
  ];

  readonly canCreate = computed(() => this.session.hasAnyRole(['ADMINISTRADOR', 'BODEGUERO']));

  readonly activeProcesses = computed(
    () =>
      this.replenishments().filter(
        (item) => item.estado !== 'CERRADA' && item.estado !== 'RECHAZADA',
      ).length,
  );

  readonly completedProcesses = computed(
    () => this.replenishments().filter((item) => item.estado === 'CERRADA').length,
  );

  readonly estimatedInvestment = computed(() =>
    this.replenishments()
      .filter((item) => item.estado !== 'RECHAZADA')
      .reduce((total, item) => total + Number(item.costo_total_estimado), 0),
  );

  readonly createForm = this.formBuilder.nonNullable.group({
    codigo_producto: ['', [Validators.required, Validators.maxLength(30)]],

    id_proveedor: [0, [Validators.required, Validators.min(1)]],

    cantidad_solicitada: ['', [Validators.required, Validators.pattern(/^\d+([.,]\d{1,3})?$/)]],

    costo_unitario_estimado: ['', [Validators.required, Validators.pattern(/^\d+([.,]\d{1,2})?$/)]],

    observacion: ['', [Validators.maxLength(1000)]],
  });

  readonly actionForm = this.formBuilder.nonNullable.group({
    observacion: ['', [Validators.maxLength(1000)]],

    motivo_rechazo: ['', [Validators.maxLength(500)]],

    cantidad_recibida: [''],
  });

  ngOnInit(): void {
    this.loadInitialData();
  }

  loadInitialData(): void {
    this.loadingBoard.set(true);
    this.errorMessage.set('');

    forkJoin({
      providers: this.api.getProviders(),

      replenishments: this.api.getReplenishments(),
    })
      .pipe(
        finalize(() => {
          this.loadingBoard.set(false);
        }),
      )
      .subscribe({
        next: (response) => {
          this.providers.set(response.providers.proveedores);

          this.replenishments.set(response.replenishments.reposiciones);
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error));
        },
      });
  }

  refresh(): void {
    this.loadInitialData();
  }

  itemsFor(state: BPMState): BPMReplenishment[] {
    return this.replenishments().filter((item) => item.estado === state);
  }

  countFor(state: BPMState): number {
    return this.itemsFor(state).length;
  }

  openCreate(): void {
    if (!this.canCreate()) {
      return;
    }

    this.errorMessage.set('');
    this.successMessage.set('');

    this.createForm.reset({
      codigo_producto: '',
      id_proveedor: 0,
      cantidad_solicitada: '',
      costo_unitario_estimado: '',
      observacion: '',
    });

    this.showCreate.set(true);
  }

  closeCreate(): void {
    if (!this.saving()) {
      this.showCreate.set(false);
    }
  }

  submitCreate(): void {
    if (this.createForm.invalid || this.saving()) {
      this.createForm.markAllAsTouched();
      return;
    }

    const raw = this.createForm.getRawValue();

    const payload: BPMCreatePayload = {
      codigo_producto: raw.codigo_producto.trim().toUpperCase(),

      id_proveedor: Number(raw.id_proveedor),

      cantidad_solicitada: this.normalizeDecimal(raw.cantidad_solicitada),

      costo_unitario_estimado: this.normalizeDecimal(raw.costo_unitario_estimado),

      observacion: raw.observacion.trim() || undefined,
    };

    this.saving.set(true);
    this.errorMessage.set('');
    this.successMessage.set('');

    this.api
      .create(payload)
      .pipe(
        finalize(() => {
          this.saving.set(false);
        }),
      )
      .subscribe({
        next: (detail) => {
          this.showCreate.set(false);
          this.selected.set(detail);

          this.successMessage.set(`Solicitud ${detail.numero_solicitud} creada como borrador.`);

          this.loadReplenishments();
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error));
        },
      });
  }

  openDetail(item: BPMReplenishment): void {
    this.loadingDetail.set(true);
    this.errorMessage.set('');

    this.api
      .getReplenishment(item.numero_solicitud)
      .pipe(
        finalize(() => {
          this.loadingDetail.set(false);
        }),
      )
      .subscribe({
        next: (detail) => {
          this.selected.set(detail);
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error));
        },
      });
  }

  closeDetail(): void {
    if (!this.saving()) {
      this.selected.set(null);
    }
  }

  actionsFor(item: BPMReplenishment): BPMAction[] {
    const role = this.session.user()?.rol;

    switch (item.estado) {
      case 'BORRADOR':
        return role === 'ADMINISTRADOR' || role === 'BODEGUERO' ? ['ENVIAR'] : [];

      case 'SOLICITADA':
        return role === 'ADMINISTRADOR' || role === 'GERENTE' ? ['APROBAR', 'RECHAZAR'] : [];

      case 'APROBADA':
        return role === 'ADMINISTRADOR' ? ['PEDIDO'] : [];

      case 'EN_PEDIDO':
        return role === 'ADMINISTRADOR' || role === 'BODEGUERO' ? ['RECIBIR'] : [];

      case 'RECIBIDA':
        return role === 'ADMINISTRADOR' || role === 'GERENTE' ? ['CERRAR'] : [];

      default:
        return [];
    }
  }

  openAction(action: BPMAction, item: BPMReplenishment): void {
    this.errorMessage.set('');
    this.successMessage.set('');

    this.actionForm.reset({
      observacion: '',
      motivo_rechazo: '',
      cantidad_recibida: action === 'RECIBIR' ? item.cantidad_solicitada : '',
    });

    this.actionDialog.set({
      action,
      item,
    });
  }

  closeAction(): void {
    if (!this.saving()) {
      this.actionDialog.set(null);
    }
  }

  submitAction(): void {
    const dialog = this.actionDialog();

    if (!dialog || this.saving()) {
      return;
    }

    const raw = this.actionForm.getRawValue();

    const observation = raw.observacion.trim();

    let operation$: Observable<BPMReplenishmentDetail>;

    switch (dialog.action) {
      case 'ENVIAR':
        operation$ = this.api.send(dialog.item.numero_solicitud, {
          observacion: observation || undefined,
        });
        break;

      case 'APROBAR':
        operation$ = this.api.approve(dialog.item.numero_solicitud, {
          observacion: observation || undefined,
        });
        break;

      case 'RECHAZAR': {
        const reason = raw.motivo_rechazo.trim();

        if (reason.length < 5 || reason.length > 500) {
          this.errorMessage.set('El motivo de rechazo debe contener entre 5 y 500 caracteres.');
          return;
        }

        const payload: BPMRejectPayload = {
          motivo_rechazo: reason,
        };

        operation$ = this.api.reject(dialog.item.numero_solicitud, payload);
        break;
      }

      case 'PEDIDO':
        operation$ = this.api.markOrder(dialog.item.numero_solicitud, {
          observacion: observation || undefined,
        });
        break;

      case 'RECIBIR': {
        const quantity = this.normalizeDecimal(raw.cantidad_recibida);

        if (!quantity || Number(quantity) <= 0) {
          this.errorMessage.set('Ingrese una cantidad recibida válida.');
          return;
        }

        const payload: BPMReceivePayload = {
          cantidad_recibida: quantity,
          observacion: observation || undefined,
        };

        operation$ = this.api.receive(dialog.item.numero_solicitud, payload);
        break;
      }

      case 'CERRAR':
        operation$ = this.api.close(dialog.item.numero_solicitud, {
          observacion: observation || undefined,
        });
        break;
    }

    this.saving.set(true);
    this.errorMessage.set('');
    this.successMessage.set('');

    operation$
      .pipe(
        finalize(() => {
          this.saving.set(false);
        }),
      )
      .subscribe({
        next: (detail) => {
          this.actionDialog.set(null);
          this.selected.set(detail);

          this.successMessage.set(
            `${detail.numero_solicitud} cambió a ${this.statusLabel(detail.estado)}.`,
          );

          this.loadReplenishments();
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error));
        },
      });
  }

  actionTitle(action: BPMAction): string {
    switch (action) {
      case 'ENVIAR':
        return 'Enviar solicitud';

      case 'APROBAR':
        return 'Aprobar reposición';

      case 'RECHAZAR':
        return 'Rechazar reposición';

      case 'PEDIDO':
        return 'Confirmar pedido';

      case 'RECIBIR':
        return 'Registrar recepción';

      case 'CERRAR':
        return 'Cerrar proceso';
    }
  }

  actionButtonLabel(action: BPMAction): string {
    switch (action) {
      case 'ENVIAR':
        return 'Enviar';

      case 'APROBAR':
        return 'Aprobar';

      case 'RECHAZAR':
        return 'Rechazar';

      case 'PEDIDO':
        return 'Marcar pedido';

      case 'RECIBIR':
        return 'Recibir';

      case 'CERRAR':
        return 'Cerrar';
    }
  }

  statusLabel(state: BPMState): string {
    switch (state) {
      case 'EN_PEDIDO':
        return 'En pedido';

      case 'SOLICITADA':
        return 'Solicitada';

      case 'APROBADA':
        return 'Aprobada';

      case 'RECHAZADA':
        return 'Rechazada';

      case 'RECIBIDA':
        return 'Recibida';

      case 'CERRADA':
        return 'Cerrada';

      default:
        return 'Borrador';
    }
  }

  statusClass(state: BPMState): string {
    return 'status-' + state.toLowerCase().replace('_', '-');
  }

  asNumber(value: string): number {
    const number = Number(value);

    return Number.isFinite(number) ? number : 0;
  }

  private loadReplenishments(): void {
    this.api.getReplenishments().subscribe({
      next: (response) => {
        this.replenishments.set(response.reposiciones);
      },

      error: (error: unknown) => {
        this.errorMessage.set(this.extractError(error));
      },
    });
  }

  private normalizeDecimal(value: string): string {
    return value.trim().replace(',', '.');
  }

  private extractError(error: unknown): string {
    if (
      error instanceof HttpErrorResponse &&
      error.error &&
      typeof error.error.message === 'string'
    ) {
      return error.error.message;
    }

    return 'No fue posible completar la operación. ' + 'Verifique la conexión con la API.';
  }
}
