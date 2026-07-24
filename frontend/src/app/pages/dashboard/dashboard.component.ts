import { CommonModule } from '@angular/common';
import { ChangeDetectorRef, Component, inject, OnInit } from '@angular/core';
import { finalize, forkJoin } from 'rxjs';

import { DashboardSummary, Product, StockAlert } from '../../core/models/sigefer.models';
import { SigeferApiService } from '../../core/services/sigefer-api.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.scss',
})
export class DashboardComponent implements OnInit {
  private readonly api = inject(SigeferApiService);
  private readonly changeDetector = inject(ChangeDetectorRef);

  dashboard: DashboardSummary | null = null;
  products: Product[] = [];
  alerts: StockAlert[] = [];

  loading = true;
  errorMessage = '';

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.loading = true;
    this.errorMessage = '';

    forkJoin({
      dashboard: this.api.getDashboard(),
      products: this.api.getLowStockProducts(),
      alerts: this.api.getPendingAlerts(),
    })
      .pipe(
        finalize(() => {
          this.loading = false;
          this.changeDetector.markForCheck();
        }),
      )
      .subscribe({
        next: ({ dashboard, products, alerts }) => {
          this.dashboard = dashboard;
          this.products = products.productos;
          this.alerts = alerts.alertas;
        },
        error: (error: unknown) => {
          console.error('Error cargando el dashboard:', error);

          this.errorMessage =
            'No fue posible cargar los indicadores. ' +
            'Verifica la conexión entre CentOS, la API Go y Oracle.';
        },
      });
  }

  stockPercentage(product: Product): number {
    if (product.stock_minimo <= 0) {
      return 100;
    }

    return Math.max(0, Math.min(100, (product.stock_disponible / product.stock_minimo) * 100));
  }

  stockSeverity(product: Product): string {
    if (product.stock_disponible <= 0) {
      return 'critical';
    }

    if (product.stock_disponible <= product.stock_minimo) {
      return 'warning';
    }

    return 'normal';
  }
}
