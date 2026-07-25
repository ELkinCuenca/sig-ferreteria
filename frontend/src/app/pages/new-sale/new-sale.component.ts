import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { finalize, forkJoin } from 'rxjs';

import { Product } from '../../core/models/sigefer.models';
import {
  Client,
  ClientListResponse,
  PaymentMethod,
  SaleCreatePayload,
  SaleCreateResponse,
} from '../../core/models/point-of-sale.models';
import { PointOfSaleService } from '../../core/services/point-of-sale.service';

interface CartLine {
  product: Product;
  quantity: number;
  discount: number;
}

@Component({
  selector: 'app-new-sale',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './new-sale.component.html',
  styleUrl: './new-sale.component.scss',
})
export class NewSaleComponent implements OnInit {
  private readonly api = inject(PointOfSaleService);

  private readonly taxRate = 0.15;

  readonly clients = signal<Client[]>([]);
  readonly products = signal<Product[]>([]);
  readonly cart = signal<CartLine[]>([]);

  readonly loading = signal(true);
  readonly submitting = signal(false);

  readonly errorMessage = signal('');
  readonly successMessage = signal('');

  readonly clientSearch = signal('');
  readonly productSearch = signal('');

  readonly selectedClientId = signal<number | null>(null);

  readonly paymentMethod = signal<PaymentMethod>('EFECTIVO');

  readonly generalDiscount = signal(0);
  readonly observation = signal('');

  readonly receipt = signal<SaleCreateResponse | null>(null);

  readonly paymentMethods: PaymentMethod[] = ['EFECTIVO', 'TARJETA', 'TRANSFERENCIA', 'MIXTO'];

  readonly filteredClients = computed(() => {
    const search = this.normalize(this.clientSearch());

    if (search === '') {
      return this.clients();
    }

    return this.clients().filter((client) => {
      const searchable = this.normalize(
        [client.identificacion, client.nombre_completo, client.tipo_identificacion].join(' '),
      );

      return searchable.includes(search);
    });
  });

  readonly filteredProducts = computed(() => {
    const search = this.normalize(this.productSearch());

    if (search === '') {
      return this.products();
    }

    return this.products().filter((product) => {
      const searchable = this.normalize(
        [product.codigo, product.nombre, product.categoria].join(' '),
      );

      return searchable.includes(search);
    });
  });

  readonly selectedClient = computed(() => {
    const selectedId = this.selectedClientId();

    if (selectedId === null) {
      return null;
    }

    return this.clients().find((client) => client.id_cliente === selectedId) ?? null;
  });

  readonly grossSubtotal = computed(() => {
    return this.roundMoney(
      this.cart().reduce((total, line) => total + line.product.precio_venta * line.quantity, 0),
    );
  });

  readonly lineDiscountTotal = computed(() => {
    return this.roundMoney(this.cart().reduce((total, line) => total + line.discount, 0));
  });

  readonly taxableSubtotal = computed(() => {
    return this.roundMoney(
      Math.max(0, this.grossSubtotal() - this.lineDiscountTotal() - this.generalDiscount()),
    );
  });

  readonly tax = computed(() => {
    return this.roundMoney(this.taxableSubtotal() * this.taxRate);
  });

  readonly estimatedTotal = computed(() => {
    return this.roundMoney(this.taxableSubtotal() + this.tax());
  });

  readonly totalQuantity = computed(() => {
    return this.cart().reduce((total, line) => total + line.quantity, 0);
  });

  readonly validationMessage = computed(() => {
    if (!this.selectedClient()) {
      return 'Seleccione un cliente.';
    }

    if (this.cart().length === 0) {
      return 'Agregue al menos un producto.';
    }

    if (this.cart().length > 50) {
      return 'La venta admite máximo 50 productos.';
    }

    for (const line of this.cart()) {
      if (!Number.isFinite(line.quantity) || line.quantity <= 0) {
        return `La cantidad de ${line.product.codigo} debe ser mayor que cero.`;
      }

      if (line.quantity > line.product.stock_disponible) {
        return `La cantidad de ${line.product.codigo} supera el stock disponible.`;
      }

      const grossLine = line.product.precio_venta * line.quantity;

      if (!Number.isFinite(line.discount) || line.discount < 0) {
        return `El descuento de ${line.product.codigo} no es válido.`;
      }

      if (line.discount > grossLine) {
        return `El descuento de ${line.product.codigo} supera el valor de la línea.`;
      }
    }

    if (!Number.isFinite(this.generalDiscount()) || this.generalDiscount() < 0) {
      return 'El descuento general no es válido.';
    }

    const subtotalAfterLines = this.grossSubtotal() - this.lineDiscountTotal();

    if (this.generalDiscount() > subtotalAfterLines) {
      return 'El descuento general supera el subtotal.';
    }

    return '';
  });

  readonly canSubmit = computed(() => {
    return !this.loading() && !this.submitting() && this.validationMessage() === '';
  });

  ngOnInit(): void {
    this.loadInitialData();
  }

  loadInitialData(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    forkJoin({
      clients: this.api.getClients(),
      products: this.api.getProducts(),
    })
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: ({
          clients,
          products,
        }: {
          clients: ClientListResponse;
          products: {
            status: string;
            total: number;
            filtro_stock_bajo: boolean;
            productos: Product[];
          };
        }) => {
          this.clients.set(clients.clientes);
          this.products.set(products.productos);

          const finalConsumer = clients.clientes.find(
            (client) => client.tipo_identificacion === 'CONSUMIDOR_FINAL',
          );

          if (finalConsumer) {
            this.selectedClientId.set(finalConsumer.id_cliente);
          }
        },
        error: (error: unknown) => {
          console.error('Error cargando punto de venta:', error);

          this.errorMessage.set(
            this.extractError(error, 'No fue posible cargar clientes y productos.'),
          );
        },
      });
  }

  updateClientSearch(event: Event): void {
    const input = event.target as HTMLInputElement;

    this.clientSearch.set(input.value);
  }

  updateProductSearch(event: Event): void {
    const input = event.target as HTMLInputElement;

    this.productSearch.set(input.value);
  }

  selectClient(client: Client): void {
    this.selectedClientId.set(client.id_cliente);
  }

  addProduct(product: Product): void {
    if (product.stock_disponible <= 0 || this.productInCart(product.codigo)) {
      return;
    }

    this.cart.update((lines) => [
      ...lines,
      {
        product,
        quantity: 1,
        discount: 0,
      },
    ]);

    this.errorMessage.set('');
  }

  removeProduct(productCode: string): void {
    this.cart.update((lines) => lines.filter((line) => line.product.codigo !== productCode));
  }

  productInCart(productCode: string): boolean {
    return this.cart().some((line) => line.product.codigo === productCode);
  }

  updateQuantity(productCode: string, event: Event): void {
    const input = event.target as HTMLInputElement;

    const value = Number(input.value);

    this.updateLine(productCode, {
      quantity: Number.isFinite(value) ? value : 0,
    });
  }

  updateLineDiscount(productCode: string, event: Event): void {
    const input = event.target as HTMLInputElement;

    const value = Number(input.value);

    this.updateLine(productCode, {
      discount: Number.isFinite(value) ? value : 0,
    });
  }

  updateGeneralDiscount(event: Event): void {
    const input = event.target as HTMLInputElement;

    const value = Number(input.value);

    this.generalDiscount.set(Number.isFinite(value) ? value : 0);
  }

  updatePaymentMethod(event: Event): void {
    const select = event.target as HTMLSelectElement;

    this.paymentMethod.set(select.value as PaymentMethod);
  }

  updateObservation(event: Event): void {
    const textarea = event.target as HTMLTextAreaElement;

    this.observation.set(textarea.value);
  }

  lineGross(line: CartLine): number {
    return this.roundMoney(line.product.precio_venta * line.quantity);
  }

  lineSubtotal(line: CartLine): number {
    return this.roundMoney(Math.max(0, this.lineGross(line) - line.discount));
  }

  submitSale(): void {
    const client = this.selectedClient();

    if (!client || !this.canSubmit()) {
      return;
    }

    const payload: SaleCreatePayload = {
      identificacion_cliente: client.identificacion,

      metodo_pago: this.paymentMethod(),

      descuento_general: this.generalDiscount().toFixed(2),

      observacion: this.observation().trim(),

      items: this.cart().map((line) => ({
        codigo_producto: line.product.codigo,

        cantidad: line.quantity.toFixed(3),

        descuento: line.discount.toFixed(2),
      })),
    };

    this.submitting.set(true);
    this.errorMessage.set('');
    this.successMessage.set('');

    this.api
      .createSale(payload)
      .pipe(
        finalize(() => {
          this.submitting.set(false);
        }),
      )
      .subscribe({
        next: (response: SaleCreateResponse) => {
          this.receipt.set(response);

          this.successMessage.set(`Venta ${response.numero_venta} registrada correctamente.`);

          this.cart.set([]);
          this.generalDiscount.set(0);
          this.observation.set('');

          this.refreshProducts();
        },
        error: (error: unknown) => {
          console.error('Error registrando venta:', error);

          this.errorMessage.set(this.extractError(error, 'No fue posible registrar la venta.'));
        },
      });
  }

  closeReceipt(): void {
    this.receipt.set(null);
  }

  private refreshProducts(): void {
    this.api.getProducts().subscribe({
      next: (response) => {
        this.products.set(response.productos);
      },
      error: (error: unknown) => {
        console.error('No se pudo actualizar el inventario:', error);
      },
    });
  }

  private updateLine(
    productCode: string,
    changes: Partial<Pick<CartLine, 'quantity' | 'discount'>>,
  ): void {
    this.cart.update((lines) =>
      lines.map((line) => {
        if (line.product.codigo !== productCode) {
          return line;
        }

        return {
          ...line,
          ...changes,
        };
      }),
    );
  }

  private roundMoney(value: number): number {
    return Math.round((value + Number.EPSILON) * 100) / 100;
  }

  private normalize(value: string): string {
    return value
      .normalize('NFD')
      .replace(/\p{Diacritic}/gu, '')
      .toLowerCase()
      .trim();
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
