package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/shopspring/decimal"

	"sigefer.local/backend/internal/models"
)

var (
	// ErrClientNotFound indica que el cliente no existe o está inactivo.
	ErrClientNotFound = errors.New(
		"cliente no encontrado o inactivo",
	)

	// ErrUserNotFound indica que el usuario no existe o está inactivo.
	ErrUserNotFound = errors.New(
		"usuario no encontrado o inactivo",
	)
)

// ProductNotFoundError identifica un producto inexistente.
type ProductNotFoundError struct {
	Code string
}

func (err *ProductNotFoundError) Error() string {
	return fmt.Sprintf(
		"producto %s no encontrado o inactivo",
		err.Code,
	)
}

// InsufficientStockError contiene información del stock insuficiente.
type InsufficientStockError struct {
	Code      string
	Requested decimal.Decimal
	Available decimal.Decimal
}

func (err *InsufficientStockError) Error() string {
	return fmt.Sprintf(
		"stock insuficiente para %s: solicitado %s, disponible %s",
		err.Code,
		err.Requested.StringFixed(3),
		err.Available.StringFixed(3),
	)
}

// InvalidDiscountError indica un descuento superior al valor permitido.
type InvalidDiscountError struct {
	Message string
}

func (err *InvalidDiscountError) Error() string {
	return err.Message
}

type lockedProduct struct {
	IDProducto      int64
	Code            string
	Name            string
	Price           decimal.Decimal
	MinimumStock    decimal.Decimal
	CurrentStock    decimal.Decimal
	ReservedStock   decimal.Decimal
	AvailableStock  decimal.Decimal
	Quantity        decimal.Decimal
	Discount        decimal.Decimal
	LineGross       decimal.Decimal
	LineNet         decimal.Decimal
	NewCurrentStock decimal.Decimal
	NewAvailable    decimal.Decimal
}

// SaleRepository administra las transacciones de venta.
type SaleRepository struct {
	db      *sql.DB
	taxRate decimal.Decimal
}

// NewSaleRepository crea el repositorio de ventas.
func NewSaleRepository(
	db *sql.DB,
	taxRate decimal.Decimal,
) *SaleRepository {
	return &SaleRepository{
		db:      db,
		taxRate: taxRate,
	}
}

// Create registra una venta completa dentro de una única transacción.
func (repository *SaleRepository) Create(
	ctx context.Context,
	request models.SaleCreateRequest,
	ipOrigin string,
) (models.SaleCreateResponse, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return models.SaleCreateResponse{}, fmt.Errorf(
			"no se pudo iniciar la transacción: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	clientID, err := findClient(
		ctx,
		tx,
		request.IdentificacionCliente,
	)
	if err != nil {
		return models.SaleCreateResponse{}, err
	}

	if err := validateUser(
		ctx,
		tx,
		request.IDUsuario,
	); err != nil {
		return models.SaleCreateResponse{}, err
	}

	sortedItems := append(
		[]models.SaleItemRequest(nil),
		request.Items...,
	)

	sort.Slice(
		sortedItems,
		func(i, j int) bool {
			return sortedItems[i].CodigoProducto <
				sortedItems[j].CodigoProducto
		},
	)

	products := make([]lockedProduct, 0, len(sortedItems))

	subtotal := decimal.Zero
	lineDiscounts := decimal.Zero

	for _, item := range sortedItems {
		product, err := lockProduct(
			ctx,
			tx,
			item,
		)
		if err != nil {
			return models.SaleCreateResponse{}, err
		}

		products = append(products, product)
		subtotal = subtotal.Add(product.LineGross)
		lineDiscounts = lineDiscounts.Add(product.Discount)
	}

	totalDiscount := lineDiscounts.Add(
		request.DescuentoGeneral,
	).Round(2)

	if totalDiscount.GreaterThan(subtotal) {
		return models.SaleCreateResponse{},
			&InvalidDiscountError{
				Message: "el descuento total supera " +
					"el subtotal de la venta",
			}
	}

	taxableBase := subtotal.Sub(totalDiscount).Round(2)

	tax := taxableBase.Mul(
		repository.taxRate,
	).Round(2)

	total := taxableBase.Add(tax).Round(2)

	saleNumber, err := nextSaleNumber(ctx, tx)
	if err != nil {
		return models.SaleCreateResponse{}, err
	}

	if err := insertSale(
		ctx,
		tx,
		clientID,
		request.IDUsuario,
		saleNumber,
		subtotal,
		totalDiscount,
		tax,
		total,
		request,
	); err != nil {
		return models.SaleCreateResponse{}, err
	}

	saleID, err := findSaleID(
		ctx,
		tx,
		saleNumber,
	)
	if err != nil {
		return models.SaleCreateResponse{}, err
	}

	itemResults := make(
		[]models.SaleItemResult,
		0,
		len(products),
	)

	alertResults := make(
		[]models.StockAlertResult,
		0,
	)

	for _, product := range products {
		if err := insertSaleDetail(
			ctx,
			tx,
			saleID,
			product,
		); err != nil {
			return models.SaleCreateResponse{}, err
		}

		if err := updateInventory(
			ctx,
			tx,
			product,
		); err != nil {
			return models.SaleCreateResponse{}, err
		}

		if err := insertMovement(
			ctx,
			tx,
			saleID,
			request.IDUsuario,
			product,
		); err != nil {
			return models.SaleCreateResponse{}, err
		}

		movementID, err := findMovementID(
			ctx,
			tx,
			saleID,
			product.IDProducto,
		)
		if err != nil {
			return models.SaleCreateResponse{}, err
		}

		alert, err := createOrUpdateAlert(
			ctx,
			tx,
			movementID,
			product,
		)
		if err != nil {
			return models.SaleCreateResponse{}, err
		}

		if alert != nil {
			alertResults = append(
				alertResults,
				*alert,
			)
		}

		itemResults = append(
			itemResults,
			models.SaleItemResult{
				CodigoProducto:  product.Code,
				NombreProducto:  product.Name,
				Cantidad:        product.Quantity.StringFixed(3),
				PrecioUnitario:  product.Price.StringFixed(2),
				Descuento:       product.Discount.StringFixed(2),
				SubtotalLinea:   product.LineNet.StringFixed(2),
				StockAnterior:   product.CurrentStock.StringFixed(3),
				StockNuevo:      product.NewCurrentStock.StringFixed(3),
				StockDisponible: product.NewAvailable.StringFixed(3),
			},
		)
	}

	if err := insertSaleAudit(
		ctx,
		tx,
		request.IDUsuario,
		saleID,
		saleNumber,
		total,
		ipOrigin,
	); err != nil {
		return models.SaleCreateResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.SaleCreateResponse{}, fmt.Errorf(
			"no se pudo confirmar la venta: %w",
			err,
		)
	}

	return models.SaleCreateResponse{
		Status:       "ok",
		IDVenta:      saleID,
		NumeroVenta:  saleNumber,
		Subtotal:     subtotal.StringFixed(2),
		Descuento:    totalDiscount.StringFixed(2),
		Impuesto:     tax.StringFixed(2),
		Total:        total.StringFixed(2),
		MetodoPago:   request.MetodoPago,
		Estado:       "COMPLETADA",
		Items:        itemResults,
		AlertasStock: alertResults,
	}, nil
}

func findClient(
	ctx context.Context,
	tx *sql.Tx,
	identification string,
) (int64, error) {
	const query = `
		SELECT ID_CLIENTE
		FROM CLIENTE
		WHERE IDENTIFICACION = :1
		  AND ESTADO = 'A'
	`

	var clientID int64

	err := tx.QueryRowContext(
		ctx,
		query,
		identification,
	).Scan(&clientID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrClientNotFound
	}

	if err != nil {
		return 0, fmt.Errorf(
			"no se pudo validar el cliente: %w",
			err,
		)
	}

	return clientID, nil
}

func validateUser(
	ctx context.Context,
	tx *sql.Tx,
	userID *int64,
) error {
	if userID == nil {
		return nil
	}

	const query = `
		SELECT COUNT(*)
		FROM USUARIO
		WHERE ID_USUARIO = :1
		  AND ESTADO = 'ACTIVO'
	`

	var total int

	if err := tx.QueryRowContext(
		ctx,
		query,
		*userID,
	).Scan(&total); err != nil {
		return fmt.Errorf(
			"no se pudo validar el usuario: %w",
			err,
		)
	}

	if total == 0 {
		return ErrUserNotFound
	}

	return nil
}

func lockProduct(
	ctx context.Context,
	tx *sql.Tx,
	item models.SaleItemRequest,
) (lockedProduct, error) {
	const query = `
		SELECT
			p.ID_PRODUCTO,
			p.NOMBRE,
			p.PRECIO_VENTA,
			p.STOCK_MINIMO,
			i.STOCK_ACTUAL,
			i.STOCK_RESERVADO,
			i.STOCK_DISPONIBLE
		FROM PRODUCTO p
		INNER JOIN INVENTARIO i
			ON i.ID_PRODUCTO = p.ID_PRODUCTO
		WHERE p.CODIGO = :1
		  AND p.ESTADO = 'A'
		FOR UPDATE OF i.STOCK_ACTUAL
	`

	var (
		productID     int64
		name          string
		price         float64
		minimumStock  float64
		currentStock  float64
		reservedStock float64
		available     float64
	)

	err := tx.QueryRowContext(
		ctx,
		query,
		item.CodigoProducto,
	).Scan(
		&productID,
		&name,
		&price,
		&minimumStock,
		&currentStock,
		&reservedStock,
		&available,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return lockedProduct{},
			&ProductNotFoundError{
				Code: item.CodigoProducto,
			}
	}

	if err != nil {
		return lockedProduct{}, fmt.Errorf(
			"no se pudo bloquear el inventario de %s: %w",
			item.CodigoProducto,
			err,
		)
	}

	priceDecimal := decimal.NewFromFloat(price)
	minimumDecimal := decimal.NewFromFloat(minimumStock)
	currentDecimal := decimal.NewFromFloat(currentStock)
	reservedDecimal := decimal.NewFromFloat(reservedStock)
	availableDecimal := decimal.NewFromFloat(available)

	if item.Cantidad.GreaterThan(availableDecimal) {
		return lockedProduct{},
			&InsufficientStockError{
				Code:      item.CodigoProducto,
				Requested: item.Cantidad,
				Available: availableDecimal,
			}
	}

	lineGross := priceDecimal.Mul(
		item.Cantidad,
	).Round(2)

	if item.Descuento.GreaterThan(lineGross) {
		return lockedProduct{},
			&InvalidDiscountError{
				Message: fmt.Sprintf(
					"el descuento del producto %s supera "+
						"el valor de la línea",
					item.CodigoProducto,
				),
			}
	}

	newCurrent := currentDecimal.Sub(
		item.Cantidad,
	).Round(3)

	newAvailable := newCurrent.Sub(
		reservedDecimal,
	).Round(3)

	return lockedProduct{
		IDProducto:      productID,
		Code:            item.CodigoProducto,
		Name:            name,
		Price:           priceDecimal,
		MinimumStock:    minimumDecimal,
		CurrentStock:    currentDecimal,
		ReservedStock:   reservedDecimal,
		AvailableStock:  availableDecimal,
		Quantity:        item.Cantidad,
		Discount:        item.Descuento,
		LineGross:       lineGross,
		LineNet:         lineGross.Sub(item.Descuento).Round(2),
		NewCurrentStock: newCurrent,
		NewAvailable:    newAvailable,
	}, nil
}

func nextSaleNumber(
	ctx context.Context,
	tx *sql.Tx,
) (string, error) {
	const query = `
		SELECT
			'VTA-' ||
			TO_CHAR(SYSTIMESTAMP, 'YYYYMMDD') ||
			'-' ||
			LPAD(SEQ_NUMERO_VENTA.NEXTVAL, 8, '0')
		FROM DUAL
	`

	var saleNumber string

	if err := tx.QueryRowContext(
		ctx,
		query,
	).Scan(&saleNumber); err != nil {
		return "", fmt.Errorf(
			"no se pudo generar el número de venta: %w",
			err,
		)
	}

	return saleNumber, nil
}

func insertSale(
	ctx context.Context,
	tx *sql.Tx,
	clientID int64,
	userID *int64,
	saleNumber string,
	subtotal decimal.Decimal,
	discount decimal.Decimal,
	tax decimal.Decimal,
	total decimal.Decimal,
	request models.SaleCreateRequest,
) error {
	const query = `
		INSERT INTO VENTA (
			ID_CLIENTE,
			ID_USUARIO,
			NUMERO_VENTA,
			SUBTOTAL,
			DESCUENTO,
			IMPUESTO,
			TOTAL,
			METODO_PAGO,
			ESTADO,
			OBSERVACION
		)
		VALUES (
			:1, :2, :3, :4, :5,
			:6, :7, :8, 'COMPLETADA', :9
		)
	`

	var userValue any
	if userID != nil {
		userValue = *userID
	}

	_, err := tx.ExecContext(
		ctx,
		query,
		clientID,
		userValue,
		saleNumber,
		subtotal.InexactFloat64(),
		discount.InexactFloat64(),
		tax.InexactFloat64(),
		total.InexactFloat64(),
		request.MetodoPago,
		request.Observacion,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo insertar la venta: %w",
			err,
		)
	}

	return nil
}

func findSaleID(
	ctx context.Context,
	tx *sql.Tx,
	saleNumber string,
) (int64, error) {
	const query = `
		SELECT ID_VENTA
		FROM VENTA
		WHERE NUMERO_VENTA = :1
	`

	var saleID int64

	if err := tx.QueryRowContext(
		ctx,
		query,
		saleNumber,
	).Scan(&saleID); err != nil {
		return 0, fmt.Errorf(
			"no se pudo recuperar la venta: %w",
			err,
		)
	}

	return saleID, nil
}

func insertSaleDetail(
	ctx context.Context,
	tx *sql.Tx,
	saleID int64,
	product lockedProduct,
) error {
	const query = `
		INSERT INTO DETALLE_VENTA (
			ID_VENTA,
			ID_PRODUCTO,
			CANTIDAD,
			PRECIO_UNITARIO,
			DESCUENTO
		)
		VALUES (:1, :2, :3, :4, :5)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		saleID,
		product.IDProducto,
		product.Quantity.InexactFloat64(),
		product.Price.InexactFloat64(),
		product.Discount.InexactFloat64(),
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo insertar el detalle de %s: %w",
			product.Code,
			err,
		)
	}

	return nil
}

func updateInventory(
	ctx context.Context,
	tx *sql.Tx,
	product lockedProduct,
) error {
	const query = `
		UPDATE INVENTARIO
		SET
			STOCK_ACTUAL = :1,
			FECHA_ULTIMO_MOVIMIENTO = CURRENT_TIMESTAMP,
			FECHA_ACTUALIZACION = CURRENT_TIMESTAMP
		WHERE ID_PRODUCTO = :2
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		product.NewCurrentStock.InexactFloat64(),
		product.IDProducto,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo actualizar el inventario de %s: %w",
			product.Code,
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"no se pudo verificar el inventario actualizado: %w",
			err,
		)
	}

	if rows != 1 {
		return fmt.Errorf(
			"se esperaba actualizar un inventario para %s",
			product.Code,
		)
	}

	return nil
}

func insertMovement(
	ctx context.Context,
	tx *sql.Tx,
	saleID int64,
	userID *int64,
	product lockedProduct,
) error {
	const query = `
		INSERT INTO MOVIMIENTO_INVENTARIO (
			ID_PRODUCTO,
			ID_USUARIO,
			ID_VENTA,
			TIPO_MOVIMIENTO,
			CANTIDAD,
			STOCK_ANTERIOR,
			STOCK_NUEVO,
			MOTIVO
		)
		VALUES (
			:1, :2, :3, 'SALIDA_VENTA',
			:4, :5, :6, :7
		)
	`

	var userValue any
	if userID != nil {
		userValue = *userID
	}

	_, err := tx.ExecContext(
		ctx,
		query,
		product.IDProducto,
		userValue,
		saleID,
		product.Quantity.InexactFloat64(),
		product.CurrentStock.InexactFloat64(),
		product.NewCurrentStock.InexactFloat64(),
		"Salida automática por venta",
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo registrar el movimiento de %s: %w",
			product.Code,
			err,
		)
	}

	return nil
}

func findMovementID(
	ctx context.Context,
	tx *sql.Tx,
	saleID int64,
	productID int64,
) (int64, error) {
	const query = `
		SELECT ID_MOVIMIENTO
		FROM MOVIMIENTO_INVENTARIO
		WHERE ID_VENTA = :1
		  AND ID_PRODUCTO = :2
		  AND TIPO_MOVIMIENTO = 'SALIDA_VENTA'
	`

	var movementID int64

	if err := tx.QueryRowContext(
		ctx,
		query,
		saleID,
		productID,
	).Scan(&movementID); err != nil {
		return 0, fmt.Errorf(
			"no se pudo recuperar el movimiento: %w",
			err,
		)
	}

	return movementID, nil
}

func createOrUpdateAlert(
	ctx context.Context,
	tx *sql.Tx,
	movementID int64,
	product lockedProduct,
) (*models.StockAlertResult, error) {
	if product.NewAvailable.GreaterThan(
		product.MinimumStock,
	) {
		return nil, nil
	}

	alertType := "STOCK_BAJO"
	if product.NewAvailable.IsZero() {
		alertType = "SIN_STOCK"
	}

	message := fmt.Sprintf(
		"El producto %s tiene stock disponible %s y mínimo %s",
		product.Code,
		product.NewAvailable.StringFixed(3),
		product.MinimumStock.StringFixed(3),
	)

	const query = `
		MERGE INTO ALERTA_STOCK destino
		USING (
			SELECT
				:1 AS ID_PRODUCTO,
				:2 AS ID_MOVIMIENTO,
				:3 AS TIPO_ALERTA,
				:4 AS STOCK_DETECTADO,
				:5 AS STOCK_MINIMO,
				:6 AS MENSAJE
			FROM DUAL
		) fuente
		ON (
			destino.ID_PRODUCTO = fuente.ID_PRODUCTO
			AND destino.ESTADO = 'PENDIENTE'
		)
		WHEN MATCHED THEN
			UPDATE SET
				destino.ID_MOVIMIENTO =
					fuente.ID_MOVIMIENTO,
				destino.TIPO_ALERTA =
					fuente.TIPO_ALERTA,
				destino.STOCK_DETECTADO =
					fuente.STOCK_DETECTADO,
				destino.STOCK_MINIMO =
					fuente.STOCK_MINIMO,
				destino.MENSAJE =
					fuente.MENSAJE,
				destino.FECHA_GENERACION =
					CURRENT_TIMESTAMP
		WHEN NOT MATCHED THEN
			INSERT (
				ID_PRODUCTO,
				ID_MOVIMIENTO,
				TIPO_ALERTA,
				STOCK_DETECTADO,
				STOCK_MINIMO,
				ESTADO,
				MENSAJE
			)
			VALUES (
				fuente.ID_PRODUCTO,
				fuente.ID_MOVIMIENTO,
				fuente.TIPO_ALERTA,
				fuente.STOCK_DETECTADO,
				fuente.STOCK_MINIMO,
				'PENDIENTE',
				fuente.MENSAJE
			)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		product.IDProducto,
		movementID,
		alertType,
		product.NewAvailable.InexactFloat64(),
		product.MinimumStock.InexactFloat64(),
		message,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo generar la alerta de %s: %w",
			product.Code,
			err,
		)
	}

	return &models.StockAlertResult{
		CodigoProducto: product.Code,
		TipoAlerta:     alertType,
		StockDetectado: product.NewAvailable.StringFixed(3),
		StockMinimo:    product.MinimumStock.StringFixed(3),
	}, nil
}

func insertSaleAudit(
	ctx context.Context,
	tx *sql.Tx,
	userID *int64,
	saleID int64,
	saleNumber string,
	total decimal.Decimal,
	ipOrigin string,
) error {
	values, err := json.Marshal(
		map[string]string{
			"numero_venta": saleNumber,
			"total":        total.StringFixed(2),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo preparar la auditoría: %w",
			err,
		)
	}

	const query = `
		INSERT INTO AUDITORIA (
			ID_USUARIO,
			TABLA_AFECTADA,
			OPERACION,
			ID_REGISTRO,
			VALORES_NUEVOS,
			IP_ORIGEN,
			ORIGEN
		)
		VALUES (
			:1,
			'VENTA',
			'INSERT',
			:2,
			:3,
			:4,
			'API REST POST /api/v1/ventas'
		)
	`

	var userValue any
	if userID != nil {
		userValue = *userID
	}

	if len(ipOrigin) > 50 {
		ipOrigin = ipOrigin[:50]
	}

	_, err = tx.ExecContext(
		ctx,
		query,
		userValue,
		fmt.Sprintf("%d", saleID),
		string(values),
		ipOrigin,
	)
	if err != nil {
		return fmt.Errorf(
			"no se pudo registrar la auditoría: %w",
			err,
		)
	}

	return nil
}

// Evita que time quede eliminado durante futuras ampliaciones del archivo.
