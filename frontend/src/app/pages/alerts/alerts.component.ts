import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { finalize, forkJoin } from 'rxjs';

import {
  AlertManagementState,
  ManagedStockAlert,
  UpdateStockAlertPayload,
  UpdateStockAlertResponse,
} from '../../core/models/sigefer.models';
import { SigeferApiService } from '../../core/services/sigefer-api.service';

type AlertFilter = 'TODAS' | AlertManagementState;

type ClosingState = 'ATENDIDA' | 'DESCARTADA';

@Component({
  selector: 'app-alerts',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './alerts.component.html',
  styleUrl: './alerts.component.scss',
})
export class AlertsComponent implements OnInit {
  private readonly api = inject(SigeferApiService);

  readonly alerts = signal<ManagedStockAlert[]>([]);
  readonly loading = signal(true);
  readonly processing = signal(false);

  readonly errorMessage = signal('');
  readonly successMessage = signal('');

  readonly selectedFilter = signal<AlertFilter>('TODAS');

  readonly selectedAlert = signal<ManagedStockAlert | null>(null);

  readonly selectedAction = signal<ClosingState>('ATENDIDA');

  readonly observation = signal('');

  readonly filteredAlerts = computed(() => {
    const filter = this.selectedFilter();

    if (filter === 'TODAS') {
      return this.alerts();
    }

    return this.alerts().filter((alert) => alert.estado === filter);
  });

  readonly pendingCount = computed(() => {
    return this.countByState('PENDIENTE');
  });

  readonly attendedCount = computed(() => {
    return this.countByState('ATENDIDA');
  });

  readonly discardedCount = computed(() => {
    return this.countByState('DESCARTADA');
  });

  readonly canSubmit = computed(() => {
    const length = Array.from(this.observation().trim()).length;

    return !this.processing() && length >= 5 && length <= 500;
  });

  ngOnInit(): void {
    this.loadAlerts();
  }

  loadAlerts(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    forkJoin({
      pending: this.api.getManagedStockAlerts('PENDIENTE'),
      attended: this.api.getManagedStockAlerts('ATENDIDA'),
      discarded: this.api.getManagedStockAlerts('DESCARTADA'),
    })
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: (responses) => {
          const alerts = [
            ...responses.pending.alertas,
            ...responses.attended.alertas,
            ...responses.discarded.alertas,
          ];

          alerts.sort((first, second) => {
            return second.fecha_generacion.localeCompare(first.fecha_generacion);
          });

          this.alerts.set(alerts);
        },
        error: (error: unknown) => {
          console.error('Error consultando alertas:', error);

          this.errorMessage.set(this.extractError(error, 'No fue posible consultar las alertas.'));
        },
      });
  }

  selectFilter(filter: AlertFilter): void {
    this.selectedFilter.set(filter);
  }

  openAction(alert: ManagedStockAlert, action: ClosingState): void {
    this.selectedAlert.set(alert);
    this.selectedAction.set(action);
    this.observation.set('');
    this.errorMessage.set('');
    this.successMessage.set('');
  }

  closeAction(): void {
    if (this.processing()) {
      return;
    }

    this.selectedAlert.set(null);
    this.observation.set('');
  }

  updateObservation(event: Event): void {
    const textarea = event.target as HTMLTextAreaElement;

    this.observation.set(textarea.value);
  }

  submitAction(): void {
    const alert = this.selectedAlert();

    if (!alert || !this.canSubmit()) {
      return;
    }

    const payload: UpdateStockAlertPayload = {
      estado: this.selectedAction(),
      observacion: this.observation().trim(),
    };

    this.processing.set(true);
    this.errorMessage.set('');
    this.successMessage.set('');

    this.api
      .updateStockAlert(alert.id_alerta, payload)
      .pipe(
        finalize(() => {
          this.processing.set(false);
        }),
      )
      .subscribe({
        next: (response: UpdateStockAlertResponse) => {
          this.selectedAlert.set(null);
          this.observation.set('');

          const action = response.estado === 'ATENDIDA' ? 'atendida' : 'descartada';

          this.successMessage.set(`La alerta ${response.id_alerta} fue ${action} correctamente.`);

          this.loadAlerts();
        },
        error: (error: unknown) => {
          console.error('Error actualizando alerta:', error);

          this.errorMessage.set(this.extractError(error, 'No fue posible actualizar la alerta.'));
        },
      });
  }

  statusClass(status: AlertManagementState): string {
    return status.toLowerCase();
  }

  typeLabel(type: string): string {
    switch (type) {
      case 'SIN_STOCK':
        return 'Sin stock';

      case 'STOCK_BAJO':
        return 'Stock bajo';

      default:
        return type;
    }
  }

  private countByState(state: AlertManagementState): number {
    return this.alerts().filter((alert) => alert.estado === state).length;
  }

  private extractError(error: unknown, fallback: string): string {
    if (
      error instanceof HttpErrorResponse &&
      error.error &&
      typeof error.error.message === 'string'
    ) {
      return error.error.message;
    }

    return fallback;
  }
}
