import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./pages/dashboard/dashboard.component').then((module) => module.DashboardComponent),
    title: 'Panel gerencial | SIGEFER',
  },
  {
    path: 'productos',
    loadComponent: () =>
      import('./pages/products/products.component').then((module) => module.ProductsComponent),
    title: 'Productos e inventario | SIGEFER',
  },
  {
    path: 'alertas',
    loadComponent: () =>
      import('./pages/alerts/alerts.component').then((module) => module.AlertsComponent),
    title: 'Alertas de stock | SIGEFER',
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
