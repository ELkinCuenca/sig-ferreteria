import { Routes } from '@angular/router';

import { roleGuard } from './core/auth/role.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () =>
      import('./pages/login/login.component').then((module) => module.LoginComponent),
    title: 'Iniciar sesión | SIGEFER',
  },
  {
    path: 'sin-permiso',
    canActivate: [roleGuard],
    loadComponent: () =>
      import('./pages/forbidden/forbidden.component').then((module) => module.ForbiddenComponent),
    title: 'Acceso no autorizado | SIGEFER',
  },
  {
    path: '',
    canActivate: [roleGuard],
    data: {
      roles: ['ADMINISTRADOR', 'GERENTE'],
    },
    loadComponent: () =>
      import('./pages/dashboard/dashboard.component').then((module) => module.DashboardComponent),
    title: 'Panel gerencial | SIGEFER',
  },
  {
    path: 'productos',
    canActivate: [roleGuard],
    data: {
      roles: ['ADMINISTRADOR', 'BODEGUERO', 'VENDEDOR'],
    },
    loadComponent: () =>
      import('./pages/products/products.component').then((module) => module.ProductsComponent),
    title: 'Productos e inventario | SIGEFER',
  },
  {
    path: 'alertas',
    canActivate: [roleGuard],
    data: {
      roles: ['ADMINISTRADOR', 'BODEGUERO'],
    },
    loadComponent: () =>
      import('./pages/alerts/alerts.component').then((module) => module.AlertsComponent),
    title: 'Alertas de stock | SIGEFER',
  },
  {
    path: 'ventas',
    canActivate: [roleGuard],
    data: {
      roles: ['ADMINISTRADOR', 'VENDEDOR', 'GERENTE'],
    },
    loadComponent: () =>
      import('./pages/sales/sales.component').then((module) => module.SalesComponent),
    title: 'Historial de ventas | SIGEFER',
  },
  {
    path: 'ventas/nueva',
    canActivate: [roleGuard],
    data: {
      roles: ['ADMINISTRADOR', 'VENDEDOR'],
    },
    loadComponent: () =>
      import('./pages/new-sale/new-sale.component').then((module) => module.NewSaleComponent),
    title: 'Registrar venta | SIGEFER',
  },
  {
    path: 'ventas/:numero',
    canActivate: [roleGuard],
    data: {
      roles: ['ADMINISTRADOR', 'VENDEDOR', 'GERENTE'],
    },
    loadComponent: () =>
      import('./pages/sales/sale-detail.component').then((module) => module.SaleDetailComponent),
    title: 'Detalle de venta | SIGEFER',
  },
  {
    path: '**',
    redirectTo: '',
  },
];
