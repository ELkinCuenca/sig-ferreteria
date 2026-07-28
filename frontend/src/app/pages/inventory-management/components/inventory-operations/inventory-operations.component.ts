import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import {
  Component,
  computed,
  EventEmitter,
  inject,
  Input,
  OnInit,
  Output,
  signal,
} from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize } from 'rxjs';

import { AuthSessionService } from '../../../../core/auth/auth-session.service';
import type {
  DecimalValue,
  InventoryAdjustmentPayload,
  InventoryMovement,
  InventoryMovementType,
  ProductApiErrorResponse,
  ProductSummary,
} from '../../../../core/models/product-admin.models';
import { ProductAdminApiService } from '../../../../core/services/product-admin-api.service';

@Component({
  selector: 'app-inventory-operations',

  standalone: true,

  imports: [CommonModule, ReactiveFormsModule],

  templateUrl: './inventory-operations.component.html',

  styleUrl: './inventory-operations.component.scss',
})
export class InventoryOperationsComponent implements OnInit {
  private readonly api = inject(ProductAdminApiService);

  private readonly session = inject(AuthSessionService);

  private readonly formBuilder = inject(FormBuilder);

  private readonly productItems = signal<ProductSummary[]>([]);

  @Input()
  set products(value: ProductSummary[]) {
    const items = value ?? [];

    this.productItems.set(items);

    const currentCode = this.adjustmentForm.controls.codigo.value;

    const currentStillValid = items.some(
      (product) => product.estado === 'A' && product.codigo === currentCode,
    );

    if (!currentStillValid) {
      const firstActiveCode = items.find((product) => product.estado === 'A')?.codigo ?? '';

      this.adjustmentForm.controls.codigo.setValue(firstActiveCode);
    }
  }

  get products(): ProductSummary[] {
    return this.productItems();
  }

  @Output()
  readonly inventoryChanged = new EventEmitter<void>();

  readonly movements = signal<InventoryMovement[]>([]);

  readonly loadingMovements = signal(false);

  readonly savingAdjustment = signal(false);

  readonly errorMessage = signal('');

  readonly successMessage = signal('');

  readonly canAdjust = computed(() => this.session.hasAnyRole(['ADMINISTRADOR', 'BODEGUERO']));

  readonly canReadMovements = computed(() =>
    this.session.hasAnyRole(['ADMINISTRADOR', 'BODEGUERO', 'GERENTE']),
  );

  readonly activeProducts = computed(() =>
    this.productItems().filter((product) => product.estado === 'A'),
  );

  selectedAdjustmentProduct(): ProductSummary | null {
    const code = this.adjustmentForm.controls.codigo.value;

    return (
      this.productItems().find((product) => product.estado === 'A' && product.codigo === code) ??
      null
    );
  }

  readonly movementTypes: {
    value: InventoryMovementType;
    label: string;
  }[] = [
    {
      value: 'ENTRADA_COMPRA',
      label: 'Entrada por compra',
    },
    {
      value: 'SALIDA_VENTA',
      label: 'Salida por venta',
    },
    {
      value: 'AJUSTE_POSITIVO',
      label: 'Ajuste positivo',
    },
    {
      value: 'AJUSTE_NEGATIVO',
      label: 'Ajuste negativo',
    },
    {
      value: 'DEVOLUCION_VENTA',
      label: 'Devolución de venta',
    },
    {
      value: 'DEVOLUCION_COMPRA',
      label: 'Devolución de compra',
    },
  ];

  readonly adjustmentForm = this.formBuilder.nonNullable.group({
    codigo: ['', [Validators.required]],

    tipo_ajuste: ['POSITIVO' as 'POSITIVO' | 'NEGATIVO', [Validators.required]],

    cantidad: [1, [Validators.required, Validators.min(0.001)]],

    motivo: ['', [Validators.required, Validators.minLength(5), Validators.maxLength(300)]],
  });

  readonly movementFilterForm = this.formBuilder.nonNullable.group({
    codigo: ['', [Validators.maxLength(30)]],

    tipo: ['' as InventoryMovementType | ''],

    limite: [50, [Validators.required, Validators.min(1), Validators.max(200)]],
  });

  ngOnInit(): void {
    if (this.canReadMovements()) {
      this.loadMovements();
    }
  }

  submitAdjustment(): void {
    this.clearMessages();

    if (!this.canAdjust()) {
      this.errorMessage.set('Tu rol no permite registrar ajustes.');

      return;
    }

    if (this.adjustmentForm.invalid) {
      this.adjustmentForm.markAllAsTouched();

      this.errorMessage.set('Revisa los datos del ajuste.');

      return;
    }

    const value = this.adjustmentForm.getRawValue();

    if (value.tipo_ajuste === 'NEGATIVO') {
      const confirmed = window.confirm(
        `¿Confirmas la disminución de ${value.cantidad} unidades del producto ${value.codigo}?`,
      );

      if (!confirmed) {
        return;
      }
    }

    const payload: InventoryAdjustmentPayload = {
      codigo: value.codigo.trim(),

      tipo_ajuste: value.tipo_ajuste,

      cantidad: value.cantidad,

      motivo: value.motivo.trim(),
    };

    this.savingAdjustment.set(true);

    this.api
      .adjustInventory(payload)
      .pipe(
        finalize(() => {
          this.savingAdjustment.set(false);
        }),
      )
      .subscribe({
        next: (response) => {
          const currentCode = value.codigo;

          this.adjustmentForm.reset({
            codigo: currentCode,

            tipo_ajuste: 'POSITIVO',

            cantidad: 1,

            motivo: '',
          });

          this.successMessage.set(
            `Ajuste registrado. Nuevo stock: ${this.formatQuantity(response.stock_nuevo)}.`,
          );

          this.loadMovements();
          this.inventoryChanged.emit();
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error, 'No fue posible registrar el ajuste.'));
        },
      });
  }

  loadMovements(): void {
    if (!this.canReadMovements()) {
      return;
    }

    this.clearMessages();

    if (this.movementFilterForm.invalid) {
      this.movementFilterForm.markAllAsTouched();

      this.errorMessage.set('Revisa los filtros del historial.');

      return;
    }

    const value = this.movementFilterForm.getRawValue();

    this.loadingMovements.set(true);

    this.api
      .getInventoryMovements({
        codigo: value.codigo.trim().toUpperCase() || undefined,

        tipo: value.tipo,

        limite: value.limite,
      })
      .pipe(
        finalize(() => {
          this.loadingMovements.set(false);
        }),
      )
      .subscribe({
        next: (response) => {
          this.movements.set(response.movimientos);
        },

        error: (error: unknown) => {
          this.errorMessage.set(
            this.extractError(error, 'No fue posible consultar los movimientos.'),
          );
        },
      });
  }

  resetMovementFilters(): void {
    this.movementFilterForm.reset({
      codigo: '',
      tipo: '',
      limite: 50,
    });

    this.loadMovements();
  }

  movementLabel(type: string): string {
    return this.movementTypes.find((item) => item.value === type)?.label ?? type;
  }

  movementClass(type: string): string {
    switch (type) {
      case 'ENTRADA_COMPRA':
      case 'AJUSTE_POSITIVO':
      case 'DEVOLUCION_VENTA':
        return 'movement-positive';

      case 'SALIDA_VENTA':
      case 'AJUSTE_NEGATIVO':
      case 'DEVOLUCION_COMPRA':
        return 'movement-negative';

      default:
        return '';
    }
  }

  formatQuantity(value: DecimalValue): string {
    const numericValue = Number(value);

    return new Intl.NumberFormat('es-EC', {
      minimumFractionDigits: 0,
      maximumFractionDigits: 3,
    }).format(Number.isFinite(numericValue) ? numericValue : 0);
  }

  formatDate(value: string): string {
    const date = new Date(value);

    if (Number.isNaN(date.getTime())) {
      return value;
    }

    return new Intl.DateTimeFormat('es-EC', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(date);
  }

  private clearMessages(): void {
    this.errorMessage.set('');
    this.successMessage.set('');
  }

  private extractError(error: unknown, fallback: string): string {
    if (error instanceof HttpErrorResponse) {
      const response = error.error as ProductApiErrorResponse | undefined;

      if (typeof response?.message === 'string' && response.message.trim() !== '') {
        return response.message;
      }
    }

    return fallback;
  }
}
