import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./pages/dashboard/dashboard.component').then((module) => module.DashboardComponent),
    title: 'Panel gerencial | SIGEFER',
  },
  {
    path: 'ventas',
    loadComponent: () =>
      import('./pages/sales/sales.component').then((module) => module.SalesComponent),
    title: 'Historial de ventas | SIGEFER',
  },
  {
    path: 'ventas/:numero',
    loadComponent: () =>
      import('./pages/sales/sale-detail.component').then((module) => module.SaleDetailComponent),
    title: 'Detalle de venta | SIGEFER',
  },
  {
    path: '**',
    redirectTo: '',
  },
];
