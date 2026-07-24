import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { finalize } from 'rxjs';

import { SaleDetail } from '../../core/models/sigefer.models';
import { SigeferApiService } from '../../core/services/sigefer-api.service';

@Component({
  selector: 'app-sale-detail',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './sale-detail.component.html',
  styleUrl: './sale-detail.component.scss',
})
export class SaleDetailComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly api = inject(SigeferApiService);

  readonly sale = signal<SaleDetail | null>(null);
  readonly loading = signal(true);
  readonly errorMessage = signal('');

  ngOnInit(): void {
    const saleNumber = this.route.snapshot.paramMap.get('numero');

    if (!saleNumber) {
      this.loading.set(false);
      this.errorMessage.set('No se proporcionó un número de venta.');
      return;
    }

    this.api
      .getSaleByNumber(saleNumber)
      .pipe(
        finalize(() => {
          this.loading.set(false);
        }),
      )
      .subscribe({
        next: (sale: SaleDetail) => {
          this.sale.set(sale);
        },
        error: (error: unknown) => {
          console.error('Error consultando la venta:', error);

          this.errorMessage.set('La venta solicitada no existe o no pudo consultarse.');
        },
      });
  }
}
