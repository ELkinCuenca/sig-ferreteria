import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize, forkJoin } from 'rxjs';

import { AuthSessionService } from '../../core/auth/auth-session.service';
import type {
  CreateProductPayload,
  DecimalValue,
  ProductApiErrorResponse,
  ProductCategory,
  ProductDetail,
  ProductStateFilter,
  ProductSummary,
  UpdateProductPayload,
} from '../../core/models/product-admin.models';
import { ProductAdminApiService } from '../../core/services/product-admin-api.service';

type ProductEditorMode = 'create' | 'detail';

@Component({
  selector: 'app-inventory-management',

  standalone: true,

  imports: [CommonModule, ReactiveFormsModule],

  templateUrl: './inventory-management.component.html',

  styleUrl: './inventory-management.component.scss',
})
export class InventoryManagementComponent implements OnInit {
  private readonly api = inject(ProductAdminApiService);

  private readonly session = inject(AuthSessionService);

  private readonly formBuilder = inject(FormBuilder);

  readonly products = signal<ProductSummary[]>([]);

  readonly categories = signal<ProductCategory[]>([]);

  readonly selectedProduct = signal<ProductDetail | null>(null);

  readonly editorMode = signal<ProductEditorMode>('detail');

  readonly loading = signal(true);

  readonly loadingDetail = signal(false);

  readonly saving = signal(false);

  readonly actionCode = signal<string | null>(null);

  readonly searchTerm = signal('');

  readonly stateFilter = signal<ProductStateFilter>('TODOS');

  readonly errorMessage = signal('');

  readonly successMessage = signal('');

  readonly canAdmin = computed(() => this.session.hasAnyRole(['ADMINISTRADOR']));

  readonly filteredProducts = computed(() => {
    const term = this.normalize(this.searchTerm());

    if (term === '') {
      return this.products();
    }

    return this.products().filter((product) => {
      const searchable = this.normalize(
        [product.codigo, product.nombre, product.categoria, product.unidad_medida].join(' '),
      );

      return searchable.includes(term);
    });
  });

  readonly activeProducts = computed(
    () => this.products().filter((product) => product.estado === 'A').length,
  );

  readonly inactiveProducts = computed(
    () => this.products().filter((product) => product.estado === 'I').length,
  );

  readonly lowStockProducts = computed(
    () =>
      this.products().filter(
        (product) =>
          product.estado === 'A' &&
          this.toNumber(product.stock_disponible) <= this.toNumber(product.stock_minimo),
      ).length,
  );

  readonly inventoryCost = computed(() =>
    this.products()
      .filter((product) => product.estado === 'A')
      .reduce(
        (total, product) =>
          total + this.toNumber(product.stock_actual) * this.toNumber(product.precio_compra),
        0,
      ),
  );

  readonly createForm = this.formBuilder.nonNullable.group({
    id_categoria: [0, [Validators.required, Validators.min(1)]],

    codigo: [
      '',
      [
        Validators.required,
        Validators.maxLength(30),
        Validators.pattern(/^[A-Z0-9][A-Z0-9._-]{1,29}$/),
      ],
    ],

    nombre: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(150)]],

    descripcion: ['', [Validators.maxLength(500)]],

    unidad_medida: ['', [Validators.required, Validators.maxLength(30)]],

    precio_compra: [0, [Validators.required, Validators.min(0)]],

    precio_venta: [0, [Validators.required, Validators.min(0)]],

    stock_minimo: [0, [Validators.required, Validators.min(0)]],

    stock_inicial: [0, [Validators.required, Validators.min(0)]],

    ubicacion: ['', [Validators.maxLength(100)]],
  });

  readonly editForm = this.formBuilder.nonNullable.group({
    id_categoria: [0, [Validators.required, Validators.min(1)]],

    nombre: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(150)]],

    descripcion: ['', [Validators.maxLength(500)]],

    unidad_medida: ['', [Validators.required, Validators.maxLength(30)]],

    precio_compra: [0, [Validators.required, Validators.min(0)]],

    precio_venta: [0, [Validators.required, Validators.min(0)]],

    stock_minimo: [0, [Validators.required, Validators.min(0)]],

    ubicacion: ['', [Validators.maxLength(100)]],
  });

  ngOnInit(): void {
    if (!this.canAdmin()) {
      this.stateFilter.set('A');
    }

    this.loadData();
  }

  loadData(): void {
    this.loading.set(true);
    this.clearMessages();

    forkJoin({
      categories: this.api.getCategories(),

      products: this.api.getProducts(this.canAdmin() ? this.stateFilter() : 'A'),
    })
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: (response) => {
          this.categories.set(response.categories.categorias);

          this.products.set(response.products.productos);

          if (this.createForm.controls.id_categoria.value === 0) {
            this.createForm.controls.id_categoria.setValue(this.categories()[0]?.id_categoria ?? 0);
          }
        },

        error: (error: unknown) => {
          this.errorMessage.set(
            this.extractError(error, 'No fue posible cargar la administración de inventario.'),
          );
        },
      });
  }

  loadProducts(): void {
    this.loading.set(true);
    this.clearMessages();

    this.api
      .getProducts(this.canAdmin() ? this.stateFilter() : 'A')
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: (response) => {
          this.products.set(response.productos);
        },

        error: (error: unknown) => {
          this.errorMessage.set(
            this.extractError(error, 'No fue posible consultar los productos.'),
          );
        },
      });
  }

  updateSearch(event: Event): void {
    const input = event.target as HTMLInputElement;

    this.searchTerm.set(input.value);
  }

  updateStateFilter(event: Event): void {
    if (!this.canAdmin()) {
      return;
    }

    const select = event.target as HTMLSelectElement;

    this.stateFilter.set(select.value as ProductStateFilter);

    this.loadProducts();
  }

  clearFilters(): void {
    this.searchTerm.set('');

    if (this.canAdmin()) {
      this.stateFilter.set('TODOS');
    }

    this.loadProducts();
  }

  startCreate(): void {
    if (!this.canAdmin()) {
      return;
    }

    this.clearMessages();
    this.editorMode.set('create');
    this.selectedProduct.set(null);

    this.createForm.reset({
      id_categoria: this.categories()[0]?.id_categoria ?? 0,

      codigo: '',
      nombre: '',
      descripcion: '',
      unidad_medida: '',
      precio_compra: 0,
      precio_venta: 0,
      stock_minimo: 0,
      stock_inicial: 0,
      ubicacion: '',
    });
  }

  openProduct(product: ProductSummary): void {
    this.clearMessages();
    this.editorMode.set('detail');
    this.loadingDetail.set(true);

    this.api
      .getProduct(product.codigo)
      .pipe(
        finalize(() => {
          this.loadingDetail.set(false);
        }),
      )
      .subscribe({
        next: (detail) => {
          this.selectedProduct.set(detail);

          this.patchEditForm(detail);
        },

        error: (error: unknown) => {
          this.errorMessage.set(
            this.extractError(error, 'No fue posible consultar el detalle del producto.'),
          );
        },
      });
  }

  submitCreate(): void {
    this.clearMessages();

    if (!this.canAdmin()) {
      this.errorMessage.set('Solo un administrador puede crear productos.');

      return;
    }

    if (this.createForm.invalid) {
      this.createForm.markAllAsTouched();

      this.errorMessage.set('Revisa los campos del nuevo producto.');

      return;
    }

    const value = this.createForm.getRawValue();

    if (value.precio_venta < value.precio_compra) {
      this.errorMessage.set('El precio de venta no puede ser menor que el precio de compra.');

      return;
    }

    const payload: CreateProductPayload = {
      id_categoria: value.id_categoria,

      codigo: value.codigo.trim().toUpperCase(),

      nombre: value.nombre.trim(),

      descripcion: value.descripcion.trim(),

      unidad_medida: value.unidad_medida.trim(),

      precio_compra: value.precio_compra,

      precio_venta: value.precio_venta,

      stock_minimo: value.stock_minimo,

      stock_inicial: value.stock_inicial,

      ubicacion: value.ubicacion.trim(),
    };

    this.saving.set(true);

    this.api
      .createProduct(payload)
      .pipe(
        finalize(() => {
          this.saving.set(false);
        }),
      )
      .subscribe({
        next: (product) => {
          this.editorMode.set('detail');

          this.selectedProduct.set(product);

          this.patchEditForm(product);

          this.loadProducts();

          this.successMessage.set(`El producto ${product.codigo} fue creado correctamente.`);
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error, 'No fue posible crear el producto.'));
        },
      });
  }

  submitEdit(): void {
    this.clearMessages();

    const product = this.selectedProduct();

    if (!this.canAdmin()) {
      this.errorMessage.set('Solo un administrador puede editar productos.');

      return;
    }

    if (!product) {
      this.errorMessage.set('No existe un producto seleccionado.');

      return;
    }

    if (this.editForm.invalid) {
      this.editForm.markAllAsTouched();

      this.errorMessage.set('Revisa los campos editables del producto.');

      return;
    }

    const value = this.editForm.getRawValue();

    if (value.precio_venta < value.precio_compra) {
      this.errorMessage.set('El precio de venta no puede ser menor que el precio de compra.');

      return;
    }

    const payload: UpdateProductPayload = {
      id_categoria: value.id_categoria,

      nombre: value.nombre.trim(),

      descripcion: value.descripcion.trim(),

      unidad_medida: value.unidad_medida.trim(),

      precio_compra: value.precio_compra,

      precio_venta: value.precio_venta,

      stock_minimo: value.stock_minimo,

      ubicacion: value.ubicacion.trim(),
    };

    this.saving.set(true);

    this.api
      .updateProduct(product.codigo, payload)
      .pipe(
        finalize(() => {
          this.saving.set(false);
        }),
      )
      .subscribe({
        next: (updatedProduct) => {
          this.selectedProduct.set(updatedProduct);

          this.patchEditForm(updatedProduct);

          this.loadProducts();

          this.successMessage.set(`El producto ${updatedProduct.codigo} fue actualizado.`);
        },

        error: (error: unknown) => {
          this.errorMessage.set(this.extractError(error, 'No fue posible actualizar el producto.'));
        },
      });
  }

  changeProductState(): void {
    this.clearMessages();

    const product = this.selectedProduct();

    if (!this.canAdmin() || !product) {
      return;
    }

    const newState = product.estado === 'A' ? 'I' : 'A';

    const action = newState === 'A' ? 'reactivar' : 'desactivar';

    const confirmed = window.confirm(
      `¿Confirmas que deseas ${action} el producto ${product.codigo}?`,
    );

    if (!confirmed) {
      return;
    }

    this.actionCode.set(product.codigo);

    this.api
      .updateProductState(product.codigo, newState)
      .pipe(
        finalize(() => {
          this.actionCode.set(null);
        }),
      )
      .subscribe({
        next: (updatedProduct) => {
          this.selectedProduct.set(updatedProduct);

          this.patchEditForm(updatedProduct);

          this.loadProducts();

          this.successMessage.set(
            `El producto ${updatedProduct.codigo} quedó ${this.stateLabel(
              updatedProduct.estado,
            ).toLowerCase()}.`,
          );
        },

        error: (error: unknown) => {
          this.errorMessage.set(
            this.extractError(error, 'No fue posible cambiar el estado del producto.'),
          );
        },
      });
  }

  cancelEditor(): void {
    this.editorMode.set('detail');
    this.selectedProduct.set(null);
    this.clearMessages();
  }

  categoryDescription(categoryId: number): string {
    return (
      this.categories().find((category) => category.id_categoria === categoryId)?.descripcion ?? ''
    );
  }

  stateLabel(state: string): string {
    return state === 'A' ? 'Activo' : 'Inactivo';
  }

  stockLabel(product: ProductSummary): string {
    if (this.toNumber(product.stock_disponible) <= 0) {
      return 'Sin stock';
    }

    if (this.toNumber(product.stock_disponible) <= this.toNumber(product.stock_minimo)) {
      return 'Stock bajo';
    }

    return 'Normal';
  }

  stockClass(product: ProductSummary): string {
    if (this.toNumber(product.stock_disponible) <= 0) {
      return 'stock-empty';
    }

    if (this.toNumber(product.stock_disponible) <= this.toNumber(product.stock_minimo)) {
      return 'stock-low';
    }

    return 'stock-normal';
  }

  formatMoney(value: DecimalValue): string {
    return new Intl.NumberFormat('es-EC', {
      style: 'currency',
      currency: 'USD',
    }).format(this.toNumber(value));
  }

  formatQuantity(value: DecimalValue): string {
    return new Intl.NumberFormat('es-EC', {
      minimumFractionDigits: 0,
      maximumFractionDigits: 3,
    }).format(this.toNumber(value));
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

  private patchEditForm(product: ProductDetail): void {
    this.editForm.reset({
      id_categoria: product.id_categoria,

      nombre: product.nombre,

      descripcion: product.descripcion ?? '',

      unidad_medida: product.unidad_medida,

      precio_compra: this.toNumber(product.precio_compra),

      precio_venta: this.toNumber(product.precio_venta),

      stock_minimo: this.toNumber(product.stock_minimo),

      ubicacion: product.ubicacion ?? '',
    });
  }

  private toNumber(value: DecimalValue): number {
    const result = Number(value);

    return Number.isFinite(result) ? result : 0;
  }

  private normalize(value: string): string {
    return value
      .normalize('NFD')
      .replace(/\p{Diacritic}/gu, '')
      .toLowerCase()
      .trim();
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
