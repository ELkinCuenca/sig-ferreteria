import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { finalize } from 'rxjs';

import { SaleListResponse, SaleSummary } from '../../core/models/sigefer.models';
import { SigeferApiService } from '../../core/services/sigefer-api.service';

@Component({
  selector: 'app-sales',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './sales.component.html',
  styleUrl: './sales.component.scss',
})
export class SalesComponent implements OnInit {
  private readonly api = inject(SigeferApiService);

  readonly sales = signal<SaleSummary[]>([]);
  readonly loading = signal(true);
  readonly errorMessage = signal('');

  ngOnInit(): void {
    this.loadSales();
  }

  loadSales(): void {
    this.loading.set(true);
    this.errorMessage.set('');

    this.api
      .getSales(100)
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: (response: SaleListResponse) => {
          this.sales.set(response.ventas);
        },
        error: (error: unknown) => {
          console.error('Error consultando las ventas:', error);

          this.errorMessage.set('No fue posible consultar el historial de ventas.');
        },
      });
  }

  completedSalesCount(): number {
    return this.sales().filter((sale) => sale.estado === 'COMPLETADA').length;
  }

  statusClass(status: string): string {
    return status.toLowerCase();
  }
}
