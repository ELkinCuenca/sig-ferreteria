package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/config"
	"sigefer.local/backend/internal/database"
	"sigefer.local/backend/internal/handlers"
	"sigefer.local/backend/internal/middleware"
	"sigefer.local/backend/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"configuración inválida: %v",
			err,
		)
	}

	taxRate, err := decimal.NewFromString(cfg.TaxRate)
	if err != nil ||
		taxRate.IsNegative() ||
		taxRate.GreaterThan(decimal.NewFromInt(1)) {
		log.Fatalf(
			"TAX_RATE debe ser un decimal entre 0 y 1",
		)
	}

	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	db, err := database.OpenOracle(rootCtx, cfg)
	if err != nil {
		log.Fatalf(
			"error conectando con Oracle: %v",
			err,
		)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf(
				"error cerrando Oracle: %v",
				err,
			)
		}
	}()

	productRepository :=
		repository.NewProductRepository(db)

	productHandler :=
		handlers.NewProductHandler(productRepository)

	clientRepository :=
		repository.NewClientRepository(db)

	clientHandler :=
		handlers.NewClientHandler(clientRepository)

	saleRepository :=
		repository.NewSaleRepository(db, taxRate)

	saleHandler :=
		handlers.NewSaleHandler(saleRepository)

	managementRepository :=
		repository.NewManagementRepository(db)

	managementHandler :=
		handlers.NewManagementHandler(managementRepository)

	bpmRepository :=
		repository.NewBPMRepository(db)

	bpmHandler :=
		handlers.NewBPMHandler(bpmRepository)

	authRepository :=
		repository.NewAuthRepository(
			db,
			cfg.SessionHours,
			cfg.MaxLoginAttempts,
		)

	authHandler :=
		handlers.NewAuthHandler(authRepository)

	requireAuthentication :=
		middleware.RequireAuthentication(authRepository)

	requireDashboardAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"GERENTE",
		)

	requireProductReadAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"BODEGUERO",
			"VENDEDOR",
			"GERENTE",
		)

	requireProductAdminAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
		)

	requireClientAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"VENDEDOR",
		)

	requireSaleWriteAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"VENDEDOR",
		)

	requireSaleReadAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"VENDEDOR",
			"GERENTE",
		)

	requireAlertAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"BODEGUERO",
		)

	requireBPMReadAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"BODEGUERO",
			"GERENTE",
		)

	requireBPMCreateAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"BODEGUERO",
		)

	requireBPMDecisionAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"GERENTE",
		)

	requireBPMOrderAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
		)

	requireBPMReceiveAccess :=
		middleware.RequireAuthentication(
			authRepository,
			"ADMINISTRADOR",
			"BODEGUERO",
		)

	router := http.NewServeMux()

	router.HandleFunc(
		"/api/v1/health",
		handlers.Health(db),
	)

	router.HandleFunc(
		"POST /api/v1/auth/login",
		authHandler.Login,
	)

	router.Handle(
		"GET /api/v1/auth/perfil",
		requireAuthentication(
			http.HandlerFunc(
				authHandler.Profile,
			),
		),
	)

	router.Handle(
		"POST /api/v1/auth/logout",
		requireAuthentication(
			http.HandlerFunc(
				authHandler.Logout,
			),
		),
	)

	router.Handle(
		"GET /api/v1/productos",
		requireProductReadAccess(
			http.HandlerFunc(
				productHandler.List,
			),
		),
	)

	router.Handle(
		"GET /api/v1/categorias",
		requireProductReadAccess(
			http.HandlerFunc(
				productHandler.ListCategories,
			),
		),
	)

	router.Handle(
		"GET /api/v1/productos/{codigo}",
		requireProductReadAccess(
			http.HandlerFunc(
				productHandler.Get,
			),
		),
	)

	router.Handle(
		"POST /api/v1/productos",
		requireProductAdminAccess(
			http.HandlerFunc(
				productHandler.Create,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/productos/{codigo}",
		requireProductAdminAccess(
			http.HandlerFunc(
				productHandler.Update,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/productos/{codigo}/estado",
		requireProductAdminAccess(
			http.HandlerFunc(
				productHandler.UpdateState,
			),
		),
	)

	router.Handle(
		"GET /api/v1/clientes",
		requireClientAccess(
			http.HandlerFunc(
				clientHandler.List,
			),
		),
	)

	router.Handle(
		"POST /api/v1/ventas",
		requireSaleWriteAccess(
			http.HandlerFunc(
				saleHandler.Create,
			),
		),
	)

	router.Handle(
		"GET /api/v1/ventas",
		requireSaleReadAccess(
			http.HandlerFunc(
				managementHandler.ListSales,
			),
		),
	)

	router.Handle(
		"GET /api/v1/ventas/{numero}",
		requireSaleReadAccess(
			http.HandlerFunc(
				managementHandler.GetSale,
			),
		),
	)

	router.Handle(
		"GET /api/v1/alertas-stock",
		requireAlertAccess(
			http.HandlerFunc(
				managementHandler.ListAlerts,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/alertas-stock/{id}",
		requireAlertAccess(
			http.HandlerFunc(
				managementHandler.UpdateAlert,
			),
		),
	)

	router.Handle(
		"GET /api/v1/dashboard/resumen",
		requireDashboardAccess(
			http.HandlerFunc(
				managementHandler.Dashboard,
			),
		),
	)

	router.Handle(
		"GET /api/v1/bpm/proveedores",
		requireBPMReadAccess(
			http.HandlerFunc(
				bpmHandler.ListProviders,
			),
		),
	)

	router.Handle(
		"GET /api/v1/bpm/reposiciones",
		requireBPMReadAccess(
			http.HandlerFunc(
				bpmHandler.List,
			),
		),
	)

	router.Handle(
		"GET /api/v1/bpm/reposiciones/{numero}",
		requireBPMReadAccess(
			http.HandlerFunc(
				bpmHandler.Get,
			),
		),
	)

	router.Handle(
		"POST /api/v1/bpm/reposiciones",
		requireBPMCreateAccess(
			http.HandlerFunc(
				bpmHandler.Create,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/bpm/reposiciones/{numero}/enviar",
		requireBPMCreateAccess(
			http.HandlerFunc(
				bpmHandler.Send,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/bpm/reposiciones/{numero}/aprobar",
		requireBPMDecisionAccess(
			http.HandlerFunc(
				bpmHandler.Approve,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/bpm/reposiciones/{numero}/rechazar",
		requireBPMDecisionAccess(
			http.HandlerFunc(
				bpmHandler.Reject,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/bpm/reposiciones/{numero}/pedido",
		requireBPMOrderAccess(
			http.HandlerFunc(
				bpmHandler.MarkOrder,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/bpm/reposiciones/{numero}/recibir",
		requireBPMReceiveAccess(
			http.HandlerFunc(
				bpmHandler.Receive,
			),
		),
	)

	router.Handle(
		"PATCH /api/v1/bpm/reposiciones/{numero}/cerrar",
		requireBPMDecisionAccess(
			http.HandlerFunc(
				bpmHandler.Close,
			),
		),
	)

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           middleware.RejectCorruptUnicode(router),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       35 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-rootCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf(
				"error apagando la API: %v",
				err,
			)
		}
	}()

	log.Printf(
		"SIGEFER API escuchando en el puerto %s",
		cfg.AppPort,
	)

	err = server.ListenAndServe()

	if err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatalf(
			"error del servidor HTTP: %v",
			err,
		)
	}

	log.Println(
		"SIGEFER API detenida correctamente",
	)
}
