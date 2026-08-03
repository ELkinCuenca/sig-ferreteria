import { CommonModule, DOCUMENT } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, inject, OnDestroy, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize, forkJoin, interval, Subscription } from 'rxjs';

import { AuthSessionService } from '../../core/auth/auth-session.service';
import {
  IoTAlert,
  IoTAlertStatus,
  IoTConfiguration,
  IoTReading,
  IoTCommunicationStatus,
  UpdateIoTConfigurationRequest,
} from '../../core/models/iot.models';
import { IoTApiService } from '../../core/services/iot-api.service';

type IoTAlertFilter = 'TODAS' | IoTAlertStatus;

type IoTMetricState = 'NORMAL' | 'ALTO' | 'BAJO' | 'SIN_DATOS';

@Component({
  selector: 'app-iot',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './iot.component.html',
  styleUrl: './iot.component.scss',
})
export class IoTComponent implements OnInit, OnDestroy {
  private readonly api = inject(IoTApiService);

  private readonly formBuilder = inject(FormBuilder);

  private readonly document = inject(DOCUMENT);

  readonly session = inject(AuthSessionService);

  private readonly subscriptions = new Subscription();

  private readonly synchronizing = signal(false);

  readonly summary = signal<import('../../core/models/iot.models').IoTSummaryResponse | null>(null);

  readonly readings = signal<IoTReading[]>([]);

  readonly alerts = signal<IoTAlert[]>([]);

  readonly selectedAlert = signal<IoTAlert | null>(null);

  readonly alertFilter = signal<IoTAlertFilter>('TODAS');

  readonly loading = signal(false);

  readonly refreshing = signal(false);

  readonly processingAlert = signal(false);

  readonly savingConfiguration = signal(false);

  readonly hasLoaded = signal(false);

  readonly errorMessage = signal('');

  readonly successMessage = signal('');

  readonly lastUpdated = signal<Date | null>(null);

  readonly canAttendAlerts = computed(() =>
    this.session.hasAnyRole(['ADMINISTRADOR', 'BODEGUERO'] as const),
  );

  readonly canConfigure = computed(() => this.session.hasAnyRole(['ADMINISTRADOR'] as const));

  readonly latestReading = computed(() => this.summary()?.ultima_lectura ?? null);

  readonly chronologicalReadings = computed(() => [...this.readings()].reverse());

  readonly filteredAlerts = computed(() => {
    const selectedFilter = this.alertFilter();

    if (selectedFilter === 'TODAS') {
      return this.alerts();
    }

    return this.alerts().filter((alert) => alert.estado === selectedFilter);
  });

  readonly pendingAlerts = computed(() =>
    this.alerts().filter((alert) => alert.estado === 'PENDIENTE'),
  );

  readonly attendedAlerts = computed(() =>
    this.alerts().filter((alert) => alert.estado === 'ATENDIDA'),
  );

  readonly criticalPendingAlerts = computed(() =>
    this.alerts().filter((alert) => alert.estado === 'PENDIENTE' && alert.severidad === 'CRITICA'),
  );

  readonly configurationForm = this.formBuilder.nonNullable.group({
    temperatura_min_c: [10, [Validators.required, Validators.min(-50), Validators.max(100)]],

    temperatura_max_c: [30, [Validators.required, Validators.min(-50), Validators.max(100)]],

    humedad_min_pct: [25, [Validators.required, Validators.min(0), Validators.max(100)]],

    humedad_max_pct: [70, [Validators.required, Validators.min(0), Validators.max(100)]],

    segundos_sin_comunicacion: [
      120,
      [Validators.required, Validators.min(30), Validators.max(86400)],
    ],
  });

  readonly alertForm = this.formBuilder.nonNullable.group({
    observacion: ['', [Validators.required, Validators.minLength(5), Validators.maxLength(500)]],
  });

  ngOnInit(): void {
    this.refresh(true);

    this.subscriptions.add(
      interval(30_000).subscribe(() => {
        if (!this.document.hidden) {
          this.refresh(false, true);
        }
      }),
    );
  }

  ngOnDestroy(): void {
    this.subscriptions.unsubscribe();
  }

  refreshNow(): void {
    this.refresh(false, false);
  }

  refresh(initial = false, silent = false): void {
    if (this.synchronizing() || this.processingAlert() || this.savingConfiguration()) {
      return;
    }

    const showInitialLoader = initial || !this.hasLoaded();

    this.synchronizing.set(true);

    if (showInitialLoader) {
      this.loading.set(true);
    } else if (!silent) {
      this.refreshing.set(true);
    }

    if (!silent) {
      this.errorMessage.set('');
      this.successMessage.set('');
    }

    forkJoin({
      summary: this.api.getSummary(),

      readings: this.api.getReadings(50),

      alerts: this.api.getAlerts('', 100),

      configuration: this.api.getConfiguration(),
    })
      .pipe(
        finalize(() => {
          this.synchronizing.set(false);
          this.loading.set(false);
          this.refreshing.set(false);
        }),
      )
      .subscribe({
        next: (result) => {
          const configuration = result.configuration.configuracion;

          this.summary.set({
            ...result.summary,
            configuracion: configuration,
          });

          this.readings.set(result.readings.lecturas);

          this.alerts.set(result.alerts.alertas);

          this.applyConfiguration(
            configuration,
            showInitialLoader || !this.configurationForm.dirty,
          );

          this.lastUpdated.set(new Date());

          this.hasLoaded.set(true);
          this.errorMessage.set('');
        },

        error: (error: unknown) => {
          if (!silent || !this.hasLoaded()) {
            this.errorMessage.set(this.readError(error, 'No fue posible cargar el monitoreo IoT.'));
          }
        },
      });
  }

  setAlertFilter(filter: IoTAlertFilter): void {
    this.alertFilter.set(filter);
  }

  openAlert(alert: IoTAlert): void {
    this.selectedAlert.set(alert);

    this.alertForm.reset({
      observacion: alert.observacion_atencion ?? '',
    });

    this.errorMessage.set('');
    this.successMessage.set('');
  }

  closeAlert(): void {
    if (this.processingAlert()) {
      return;
    }

    this.selectedAlert.set(null);

    this.alertForm.reset({
      observacion: '',
    });
  }

  canSubmitAlert(): boolean {
    const alert = this.selectedAlert();

    return (
      alert !== null &&
      alert.estado === 'PENDIENTE' &&
      this.canAttendAlerts() &&
      this.alertForm.valid &&
      !this.processingAlert()
    );
  }

  attendSelectedAlert(): void {
    const alert = this.selectedAlert();

    if (alert === null || alert.estado !== 'PENDIENTE' || !this.canAttendAlerts()) {
      return;
    }

    this.alertForm.markAllAsTouched();

    if (this.alertForm.invalid) {
      this.errorMessage.set('La observación debe contener entre 5 y 500 caracteres.');
      return;
    }

    const observation = this.alertForm.controls.observacion.value.trim();

    if (observation.length < 5 || observation.length > 500) {
      this.errorMessage.set('La observación debe contener entre 5 y 500 caracteres.');
      return;
    }

    this.processingAlert.set(true);
    this.errorMessage.set('');
    this.successMessage.set('');

    this.api
      .attendAlert(alert.id_alerta_iot, {
        observacion: observation,
      })
      .pipe(
        finalize(() => {
          this.processingAlert.set(false);
        }),
      )
      .subscribe({
        next: (result) => {
          const updatedAlert: IoTAlert = {
            ...alert,
            ...result,

            fecha_actualizacion: result.fecha_atencion,
          };

          this.alerts.update((currentAlerts) =>
            currentAlerts.map((currentAlert) =>
              currentAlert.id_alerta_iot === updatedAlert.id_alerta_iot
                ? updatedAlert
                : currentAlert,
            ),
          );

          this.summary.update((currentSummary) => {
            if (currentSummary === null) {
              return null;
            }

            return {
              ...currentSummary,

              alertas_pendientes: Math.max(0, currentSummary.alertas_pendientes - 1),
            };
          });

          this.selectedAlert.set(updatedAlert);

          this.alertForm.reset({
            observacion: updatedAlert.observacion_atencion ?? '',
          });

          this.successMessage.set(
            `La alerta ${updatedAlert.id_alerta_iot} fue atendida correctamente.`,
          );
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.readError(error, 'No fue posible atender la alerta IoT.'));
        },
      });
  }

  saveConfiguration(): void {
    if (!this.canConfigure()) {
      this.errorMessage.set('El usuario no tiene permiso para modificar la configuración IoT.');
      return;
    }

    this.configurationForm.markAllAsTouched();

    if (this.configurationForm.invalid) {
      this.errorMessage.set('Revise los valores de configuración antes de guardar.');
      return;
    }

    const values = this.configurationForm.getRawValue();

    if (values.temperatura_min_c >= values.temperatura_max_c) {
      this.errorMessage.set('La temperatura mínima debe ser menor que la temperatura máxima.');
      return;
    }

    if (values.humedad_min_pct >= values.humedad_max_pct) {
      this.errorMessage.set('La humedad mínima debe ser menor que la humedad máxima.');
      return;
    }

    const payload: UpdateIoTConfigurationRequest = {
      temperatura_min_c: values.temperatura_min_c,

      temperatura_max_c: values.temperatura_max_c,

      humedad_min_pct: values.humedad_min_pct,

      humedad_max_pct: values.humedad_max_pct,

      segundos_sin_comunicacion: values.segundos_sin_comunicacion,
    };

    this.savingConfiguration.set(true);

    this.errorMessage.set('');
    this.successMessage.set('');

    this.api
      .updateConfiguration(payload)
      .pipe(
        finalize(() => {
          this.savingConfiguration.set(false);
        }),
      )
      .subscribe({
        next: (response) => {
          const configuration = response.configuracion;

          this.applyConfiguration(configuration, true);

          this.summary.update((currentSummary) => {
            if (currentSummary === null) {
              return null;
            }

            return {
              ...currentSummary,

              configuracion: configuration,
            };
          });

          this.successMessage.set('Los umbrales ambientales fueron actualizados correctamente.');
        },

        error: (error: unknown) => {
          this.errorMessage.set(
            this.readError(error, 'No fue posible actualizar la configuración IoT.'),
          );
        },
      });
  }

  resetConfigurationForm(): void {
    const configuration = this.summary()?.configuracion;

    if (configuration === undefined) {
      return;
    }

    this.applyConfiguration(configuration, true);

    this.errorMessage.set('');
    this.successMessage.set('');
  }

  private applyConfiguration(configuration: IoTConfiguration, force: boolean): void {
    if (!force) {
      return;
    }

    this.configurationForm.setValue(
      {
        temperatura_min_c: configuration.temperatura_min_c,

        temperatura_max_c: configuration.temperatura_max_c,

        humedad_min_pct: configuration.humedad_min_pct,

        humedad_max_pct: configuration.humedad_max_pct,

        segundos_sin_comunicacion: configuration.segundos_sin_comunicacion,
      },
      {
        emitEvent: false,
      },
    );

    this.configurationForm.markAsPristine();

    this.configurationForm.markAsUntouched();
  }

  metricState(value: number | null, minimum: number, maximum: number): IoTMetricState {
    if (value === null) {
      return 'SIN_DATOS';
    }

    if (value < minimum) {
      return 'BAJO';
    }

    if (value > maximum) {
      return 'ALTO';
    }

    return 'NORMAL';
  }

  metricStateLabel(state: IoTMetricState): string {
    const labels: Record<IoTMetricState, string> = {
      NORMAL: 'Normal',
      ALTO: 'Alto',
      BAJO: 'Bajo',
      SIN_DATOS: 'Sin datos',
    };

    return labels[state];
  }

  metricClass(state: IoTMetricState): string {
    return `metric-${state.toLowerCase()}`;
  }

  communicationLabel(status: IoTCommunicationStatus): string {
    const labels: Record<IoTCommunicationStatus, string> = {
      EN_LINEA: 'En línea',

      SIN_COMUNICACION: 'Sin comunicación',

      SIN_DATOS: 'Sin datos',
    };

    return labels[status];
  }

  communicationClass(status: IoTCommunicationStatus): string {
    return `communication-${status.toLowerCase()}`;
  }

  alertTypeLabel(type: string): string {
    const labels: Record<string, string> = {
      TEMPERATURA_ALTA: 'Temperatura alta',

      TEMPERATURA_BAJA: 'Temperatura baja',

      HUMEDAD_ALTA: 'Humedad alta',

      HUMEDAD_BAJA: 'Humedad baja',

      SIN_COMUNICACION: 'Sin comunicación',

      SENSOR_FALLIDO: 'Fallo del sensor',
    };

    return labels[type] ?? type;
  }

  alertSeverityClass(severity: string): string {
    return `severity-${severity.toLowerCase()}`;
  }

  alertStatusClass(status: IoTAlertStatus): string {
    return `alert-${status.toLowerCase()}`;
  }

  signalQuality(rssi: number | null): string {
    if (rssi === null) {
      return 'Sin datos';
    }

    if (rssi >= -55) {
      return 'Excelente';
    }

    if (rssi >= -67) {
      return 'Buena';
    }

    if (rssi >= -75) {
      return 'Regular';
    }

    return 'Débil';
  }

  signalClass(rssi: number | null): string {
    if (rssi === null) {
      return 'signal-none';
    }

    if (rssi >= -55) {
      return 'signal-excellent';
    }

    if (rssi >= -67) {
      return 'signal-good';
    }

    if (rssi >= -75) {
      return 'signal-regular';
    }

    return 'signal-weak';
  }

  signalPercentage(rssi: number | null): number {
    if (rssi === null) {
      return 0;
    }

    const minimum = -100;
    const maximum = -30;

    return Math.round(Math.min(100, Math.max(0, ((rssi - minimum) / (maximum - minimum)) * 100)));
  }

  formatValue(value: number | null, digits = 1): string {
    if (value === null) {
      return '—';
    }

    return new Intl.NumberFormat('es-EC', {
      minimumFractionDigits: digits,

      maximumFractionDigits: digits,
    }).format(value);
  }

  dateLabel(value: string | Date | null | undefined): string {
    if (value === null || value === undefined || value === '') {
      return 'Sin registro';
    }

    const date = value instanceof Date ? value : new Date(value);

    if (Number.isNaN(date.getTime())) {
      return String(value);
    }

    return new Intl.DateTimeFormat('es-EC', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(date);
  }

  uptimeLabel(totalSeconds: number): string {
    const safeSeconds = Math.max(0, Math.floor(totalSeconds));

    const days = Math.floor(safeSeconds / 86400);

    const hours = Math.floor((safeSeconds % 86400) / 3600);

    const minutes = Math.floor((safeSeconds % 3600) / 60);

    if (days > 0) {
      return `${days} d ${hours} h`;
    }

    if (hours > 0) {
      return `${hours} h ${minutes} min`;
    }

    return `${minutes} min`;
  }

  trackReading(_index: number, reading: IoTReading): number {
    return reading.id_lectura;
  }

  trackAlert(_index: number, alert: IoTAlert): number {
    return alert.id_alerta_iot;
  }

  private readError(error: unknown, fallback: string): string {
    if (error instanceof HttpErrorResponse) {
      const responseBody = error.error as
        | {
            message?: unknown;
          }
        | null
        | undefined;

      if (
        responseBody !== null &&
        responseBody !== undefined &&
        typeof responseBody.message === 'string' &&
        responseBody.message.trim() !== ''
      ) {
        return responseBody.message;
      }

      if (typeof error.message === 'string' && error.message.trim() !== '') {
        return error.message;
      }
    }

    return fallback;
  }
}
