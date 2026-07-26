import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { finalize } from 'rxjs';

import { ProductCategory } from '../../core/models/product-admin.models';
import { ProductAdminApiService } from '../../core/services/product-admin-api.service';

@Component({
  selector: 'app-inventory-management',

  standalone: true,

  imports: [CommonModule],

  templateUrl: './inventory-management.component.html',

  styleUrl: './inventory-management.component.scss',
})
export class InventoryManagementComponent implements OnInit {
  private readonly api = inject(ProductAdminApiService);

  readonly categories = signal<ProductCategory[]>([]);

  readonly loading = signal(true);
  readonly errorMessage = signal('');

  ngOnInit(): void {
    this.loadCategories();
  }

  loadCategories(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    this.api
      .getCategories()
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: (response) => {
          this.categories.set(response.categorias);
        },

        error: (error: unknown) => {
          console.error('Error consultando categorías:', error);

          this.errorMessage.set('No fue posible consultar las categorías.');
        },
      });
  }
}
