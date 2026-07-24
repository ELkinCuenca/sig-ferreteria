import { CommonModule } from '@angular/common';
import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { finalize } from 'rxjs';

import { Product, ProductListResponse } from '../../core/models/sigefer.models';
import { SigeferApiService } from '../../core/services/sigefer-api.service';

@Component({
  selector: 'app-products',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './products.component.html',
  styleUrl: './products.component.scss',
})
export class ProductsComponent implements OnInit {
  private readonly api = inject(SigeferApiService);

  readonly products = signal<Product[]>([]);
  readonly loading = signal(true);
  readonly errorMessage = signal('');

  readonly searchTerm = signal('');
  readonly statusFilter = signal('TODOS');
  readonly selectedProduct = signal<Product | null>(null);

  readonly statuses = computed(() => {
    return Array.from(new Set(this.products().map((product) => product.estado_stock))).sort();
  });

  readonly filteredProducts = computed(() => {
    const term = this.normalize(this.searchTerm());
    const status = this.statusFilter();

    return this.products().filter((product) => {
      const searchableText = this.normalize(
        [product.codigo, product.nombre, product.categoria, product.unidad_medida].join(' '),
      );

      const matchesSearch = term === '' || searchableText.includes(term);

      const matchesStatus = status === 'TODOS' || product.estado_stock === status;

      return matchesSearch && matchesStatus;
    });
  });

  readonly lowStockCount = computed(() => {
    return this.products().filter((product) => product.stock_disponible <= product.stock_minimo)
      .length;
  });

  readonly totalUnits = computed(() => {
    return this.products().reduce((total, product) => total + product.stock_actual, 0);
  });

  readonly inventoryCost = computed(() => {
    return this.products().reduce(
      (total, product) => total + product.stock_actual * product.precio_compra,
      0,
    );
  });

  ngOnInit(): void {
    this.loadProducts();
  }

  loadProducts(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    this.api
      .getProducts()
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: (response: ProductListResponse) => {
          this.products.set(response.productos);
        },
        error: (error: unknown) => {
          console.error('Error consultando productos:', error);

          this.errorMessage.set('No fue posible consultar el inventario.');
        },
      });
  }

  updateSearch(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.searchTerm.set(input.value);
  }

  updateStatus(event: Event): void {
    const select = event.target as HTMLSelectElement;
    this.statusFilter.set(select.value);
  }

  clearFilters(): void {
    this.searchTerm.set('');
    this.statusFilter.set('TODOS');
  }

  openProduct(product: Product): void {
    this.selectedProduct.set(product);
  }

  closeProduct(): void {
    this.selectedProduct.set(null);
  }

  stockPercentage(product: Product): number {
    if (product.stock_minimo <= 0) {
      return 100;
    }

    return Math.max(0, Math.min(100, (product.stock_disponible / product.stock_minimo) * 100));
  }

  stockClass(product: Product): string {
    if (product.stock_disponible <= 0) {
      return 'agotado';
    }

    if (product.stock_disponible <= product.stock_minimo) {
      return 'bajo';
    }

    return 'normal';
  }

  statusClass(status: string): string {
    return status.toLowerCase().replaceAll('_', '-');
  }

  private normalize(value: string): string {
    return value
      .normalize('NFD')
      .replace(/\p{Diacritic}/gu, '')
      .toLowerCase()
      .trim();
  }
}
