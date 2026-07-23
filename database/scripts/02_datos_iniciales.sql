SET ECHO ON
SET SQLBLANKLINES ON
SET FEEDBACK ON
SET SERVEROUTPUT ON
SET DEFINE OFF
SET PAGESIZE 100
SET LINESIZE 200

WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK
WHENEVER OSERROR EXIT FAILURE

PROMPT ============================================================
PROMPT SIGEFER - CARGA DE DATOS INICIALES
PROMPT Esquema esperado: SIGEFER_APP
PROMPT ============================================================

DECLARE
    v_usuario VARCHAR2(128);
BEGIN
    SELECT USER
    INTO v_usuario
    FROM DUAL;

    IF v_usuario <> 'SIGEFER_APP' THEN
        RAISE_APPLICATION_ERROR(
            -20001,
            'El script debe ejecutarse con SIGEFER_APP'
        );
    END IF;

    DBMS_OUTPUT.PUT_LINE(
        'Esquema validado: ' || v_usuario
    );
END;
/

SAVEPOINT INICIO_CARGA;

PROMPT ============================================================
PROMPT 1. ROLES
PROMPT ============================================================

MERGE INTO ROL destino
USING (
    SELECT
        'ADMINISTRADOR' AS NOMBRE,
        'Administración completa del sistema SIGEFER' AS DESCRIPCION
    FROM DUAL

    UNION ALL

    SELECT
        'GERENTE',
        'Consulta de indicadores y apoyo a decisiones gerenciales'
    FROM DUAL

    UNION ALL

    SELECT
        'VENDEDOR',
        'Registro de clientes y operaciones de venta'
    FROM DUAL

    UNION ALL

    SELECT
        'BODEGUERO',
        'Gestión de inventario, entradas y reposición'
    FROM DUAL
) fuente
ON (
    destino.NOMBRE = fuente.NOMBRE
)
WHEN MATCHED THEN
    UPDATE SET
        destino.DESCRIPCION = fuente.DESCRIPCION,
        destino.ESTADO = 'A',
        destino.FECHA_ACTUALIZACION = CURRENT_TIMESTAMP
WHEN NOT MATCHED THEN
    INSERT (
        NOMBRE,
        DESCRIPCION,
        ESTADO
    )
    VALUES (
        fuente.NOMBRE,
        fuente.DESCRIPCION,
        'A'
    );

PROMPT ============================================================
PROMPT 2. CATEGORIAS
PROMPT ============================================================

MERGE INTO CATEGORIA destino
USING (
    SELECT
        'HERRAMIENTAS MANUALES' AS NOMBRE,
        'Herramientas de operación manual' AS DESCRIPCION
    FROM DUAL

    UNION ALL

    SELECT
        'HERRAMIENTAS ELECTRICAS',
        'Equipos y herramientas alimentadas con electricidad'
    FROM DUAL

    UNION ALL

    SELECT
        'PINTURAS',
        'Pinturas, recubrimientos y accesorios'
    FROM DUAL

    UNION ALL

    SELECT
        'ELECTRICIDAD',
        'Cables y accesorios para instalaciones eléctricas'
    FROM DUAL

    UNION ALL

    SELECT
        'PLOMERIA',
        'Tuberías y accesorios para instalaciones sanitarias'
    FROM DUAL

    UNION ALL

    SELECT
        'FIJACIONES',
        'Tornillos, clavos, pernos y elementos de fijación'
    FROM DUAL

    UNION ALL

    SELECT
        'SEGURIDAD INDUSTRIAL',
        'Equipos de protección personal'
    FROM DUAL
) fuente
ON (
    destino.NOMBRE = fuente.NOMBRE
)
WHEN MATCHED THEN
    UPDATE SET
        destino.DESCRIPCION = fuente.DESCRIPCION,
        destino.ESTADO = 'A',
        destino.FECHA_ACTUALIZACION = CURRENT_TIMESTAMP
WHEN NOT MATCHED THEN
    INSERT (
        NOMBRE,
        DESCRIPCION,
        ESTADO
    )
    VALUES (
        fuente.NOMBRE,
        fuente.DESCRIPCION,
        'A'
    );

PROMPT ============================================================
PROMPT 3. PROVEEDORES
PROMPT ============================================================

MERGE INTO PROVEEDOR destino
USING (
    SELECT
        '0790001001001' AS RUC,
        'DISTRIBUIDORA FERREMACHALA S.A.' AS RAZON_SOCIAL,
        'María Torres' AS NOMBRE_CONTACTO,
        '0991112233' AS TELEFONO,
        'ventas@ferremachala.demo' AS CORREO,
        'Machala, El Oro' AS DIRECCION
    FROM DUAL

    UNION ALL

    SELECT
        '0990002002001',
        'SUMINISTROS INDUSTRIALES DEL PACIFICO S.A.',
        'Carlos Mendoza',
        '0982223344',
        'comercial@pacifico.demo',
        'Guayaquil, Guayas'
    FROM DUAL

    UNION ALL

    SELECT
        '0790003003001',
        'PINTURAS Y ACABADOS DEL SUR C.L.',
        'Andrea Romero',
        '0973334455',
        'pedidos@pinturassur.demo',
        'Pasaje, El Oro'
    FROM DUAL
) fuente
ON (
    destino.RUC = fuente.RUC
)
WHEN MATCHED THEN
    UPDATE SET
        destino.RAZON_SOCIAL = fuente.RAZON_SOCIAL,
        destino.NOMBRE_CONTACTO = fuente.NOMBRE_CONTACTO,
        destino.TELEFONO = fuente.TELEFONO,
        destino.CORREO = fuente.CORREO,
        destino.DIRECCION = fuente.DIRECCION,
        destino.ESTADO = 'A',
        destino.FECHA_ACTUALIZACION = CURRENT_TIMESTAMP
WHEN NOT MATCHED THEN
    INSERT (
        RUC,
        RAZON_SOCIAL,
        NOMBRE_CONTACTO,
        TELEFONO,
        CORREO,
        DIRECCION,
        ESTADO
    )
    VALUES (
        fuente.RUC,
        fuente.RAZON_SOCIAL,
        fuente.NOMBRE_CONTACTO,
        fuente.TELEFONO,
        fuente.CORREO,
        fuente.DIRECCION,
        'A'
    );

PROMPT ============================================================
PROMPT 4. CLIENTES
PROMPT ============================================================

MERGE INTO CLIENTE destino
USING (
    SELECT
        'CONSUMIDOR_FINAL' AS TIPO_IDENTIFICACION,
        '9999999999999' AS IDENTIFICACION,
        CAST(NULL AS VARCHAR2(100)) AS NOMBRES,
        CAST(NULL AS VARCHAR2(100)) AS APELLIDOS,
        'CONSUMIDOR FINAL' AS RAZON_SOCIAL,
        CAST(NULL AS VARCHAR2(30)) AS TELEFONO,
        CAST(NULL AS VARCHAR2(150)) AS CORREO,
        CAST(NULL AS VARCHAR2(300)) AS DIRECCION
    FROM DUAL

    UNION ALL

    SELECT
        'CEDULA',
        '0700000001',
        'Juan Carlos',
        'Ramírez López',
        CAST(NULL AS VARCHAR2(180)),
        '0984445566',
        'juan.ramirez@correo.demo',
        'Machala, El Oro'
    FROM DUAL

    UNION ALL

    SELECT
        'RUC',
        '0790004004001',
        CAST(NULL AS VARCHAR2(100)),
        CAST(NULL AS VARCHAR2(100)),
        'CONSTRUCTORA ORENSE DEMO C.L.',
        '0995556677',
        'compras@constructoraorense.demo',
        'Machala, El Oro'
    FROM DUAL
) fuente
ON (
    destino.IDENTIFICACION = fuente.IDENTIFICACION
)
WHEN MATCHED THEN
    UPDATE SET
        destino.TIPO_IDENTIFICACION = fuente.TIPO_IDENTIFICACION,
        destino.NOMBRES = fuente.NOMBRES,
        destino.APELLIDOS = fuente.APELLIDOS,
        destino.RAZON_SOCIAL = fuente.RAZON_SOCIAL,
        destino.TELEFONO = fuente.TELEFONO,
        destino.CORREO = fuente.CORREO,
        destino.DIRECCION = fuente.DIRECCION,
        destino.ESTADO = 'A',
        destino.FECHA_ACTUALIZACION = CURRENT_TIMESTAMP
WHEN NOT MATCHED THEN
    INSERT (
        TIPO_IDENTIFICACION,
        IDENTIFICACION,
        NOMBRES,
        APELLIDOS,
        RAZON_SOCIAL,
        TELEFONO,
        CORREO,
        DIRECCION,
        ESTADO
    )
    VALUES (
        fuente.TIPO_IDENTIFICACION,
        fuente.IDENTIFICACION,
        fuente.NOMBRES,
        fuente.APELLIDOS,
        fuente.RAZON_SOCIAL,
        fuente.TELEFONO,
        fuente.CORREO,
        fuente.DIRECCION,
        'A'
    );

PROMPT ============================================================
PROMPT 5. PRODUCTOS
PROMPT ============================================================

MERGE INTO PRODUCTO destino
USING (
    SELECT
        datos.CODIGO,
        datos.NOMBRE,
        datos.DESCRIPCION,
        datos.UNIDAD_MEDIDA,
        datos.PRECIO_COMPRA,
        datos.PRECIO_VENTA,
        datos.STOCK_MINIMO,
        categoria.ID_CATEGORIA
    FROM (
        SELECT
            'MART-001' AS CODIGO,
            'Martillo de carpintero 16 oz' AS NOMBRE,
            'Martillo con mango ergonómico' AS DESCRIPCION,
            'UNIDAD' AS UNIDAD_MEDIDA,
            6.50 AS PRECIO_COMPRA,
            9.50 AS PRECIO_VENTA,
            8 AS STOCK_MINIMO,
            'HERRAMIENTAS MANUALES' AS CATEGORIA
        FROM DUAL

        UNION ALL

        SELECT
            'DEST-001',
            'Destornillador plano 6 pulgadas',
            'Destornillador de acero con mango aislado',
            'UNIDAD',
            2.20,
            3.75,
            12,
            'HERRAMIENTAS MANUALES'
        FROM DUAL

        UNION ALL

        SELECT
            'TALA-001',
            'Taladro percutor de 1/2 pulgada',
            'Taladro eléctrico para trabajo general',
            'UNIDAD',
            45.00,
            62.00,
            3,
            'HERRAMIENTAS ELECTRICAS'
        FROM DUAL

        UNION ALL

        SELECT
            'DISC-001',
            'Disco de corte para metal 4.5 pulgadas',
            'Disco abrasivo para corte de metal',
            'UNIDAD',
            1.10,
            1.75,
            20,
            'HERRAMIENTAS ELECTRICAS'
        FROM DUAL

        UNION ALL

        SELECT
            'PINT-001',
            'Pintura látex blanca un galón',
            'Pintura para interiores y exteriores',
            'GALON',
            11.50,
            16.50,
            6,
            'PINTURAS'
        FROM DUAL

        UNION ALL

        SELECT
            'CABL-001',
            'Cable flexible número 12',
            'Cable de cobre para instalaciones eléctricas',
            'METRO',
            0.48,
            0.70,
            100,
            'ELECTRICIDAD'
        FROM DUAL

        UNION ALL

        SELECT
            'TUBO-001',
            'Tubo PVC 1/2 pulgada x 3 metros',
            'Tubo para instalaciones de agua',
            'UNIDAD',
            2.70,
            4.10,
            15,
            'PLOMERIA'
        FROM DUAL

        UNION ALL

        SELECT
            'TORN-001',
            'Tornillo drywall de 1 pulgada',
            'Tornillo para fijación en paneles',
            'UNIDAD',
            0.03,
            0.06,
            500,
            'FIJACIONES'
        FROM DUAL

        UNION ALL

        SELECT
            'GUAN-001',
            'Guantes de nitrilo',
            'Guantes de protección para trabajo general',
            'PAR',
            1.80,
            3.00,
            25,
            'SEGURIDAD INDUSTRIAL'
        FROM DUAL

        UNION ALL

        SELECT
            'CASC-001',
            'Casco de seguridad industrial',
            'Casco de protección ajustable',
            'UNIDAD',
            6.50,
            9.80,
            10,
            'SEGURIDAD INDUSTRIAL'
        FROM DUAL
    ) datos
    INNER JOIN CATEGORIA categoria
        ON categoria.NOMBRE = datos.CATEGORIA
) fuente
ON (
    destino.CODIGO = fuente.CODIGO
)
WHEN MATCHED THEN
    UPDATE SET
        destino.ID_CATEGORIA = fuente.ID_CATEGORIA,
        destino.NOMBRE = fuente.NOMBRE,
        destino.DESCRIPCION = fuente.DESCRIPCION,
        destino.UNIDAD_MEDIDA = fuente.UNIDAD_MEDIDA,
        destino.PRECIO_COMPRA = fuente.PRECIO_COMPRA,
        destino.PRECIO_VENTA = fuente.PRECIO_VENTA,
        destino.STOCK_MINIMO = fuente.STOCK_MINIMO,
        destino.ESTADO = 'A',
        destino.FECHA_ACTUALIZACION = CURRENT_TIMESTAMP
WHEN NOT MATCHED THEN
    INSERT (
        ID_CATEGORIA,
        CODIGO,
        NOMBRE,
        DESCRIPCION,
        UNIDAD_MEDIDA,
        PRECIO_COMPRA,
        PRECIO_VENTA,
        STOCK_MINIMO,
        ESTADO
    )
    VALUES (
        fuente.ID_CATEGORIA,
        fuente.CODIGO,
        fuente.NOMBRE,
        fuente.DESCRIPCION,
        fuente.UNIDAD_MEDIDA,
        fuente.PRECIO_COMPRA,
        fuente.PRECIO_VENTA,
        fuente.STOCK_MINIMO,
        'A'
    );

PROMPT ============================================================
PROMPT 6. INVENTARIO
PROMPT ============================================================

MERGE INTO INVENTARIO destino
USING (
    SELECT
        producto.ID_PRODUCTO,
        datos.STOCK_ACTUAL,
        datos.STOCK_RESERVADO,
        datos.UBICACION
    FROM (
        SELECT
            'MART-001' AS CODIGO,
            5 AS STOCK_ACTUAL,
            0 AS STOCK_RESERVADO,
            'PERCHA A1' AS UBICACION
        FROM DUAL

        UNION ALL

        SELECT 'DEST-001', 30, 2, 'PERCHA A2' FROM DUAL
        UNION ALL
        SELECT 'TALA-001', 2, 0, 'VITRINA E1' FROM DUAL
        UNION ALL
        SELECT 'DISC-001', 60, 5, 'PERCHA E2' FROM DUAL
        UNION ALL
        SELECT 'PINT-001', 4, 0, 'BODEGA P1' FROM DUAL
        UNION ALL
        SELECT 'CABL-001', 250, 20, 'BOBINA C1' FROM DUAL
        UNION ALL
        SELECT 'TUBO-001', 10, 0, 'BODEGA T1' FROM DUAL
        UNION ALL
        SELECT 'TORN-001', 1200, 100, 'GAVETA F1' FROM DUAL
        UNION ALL
        SELECT 'GUAN-001', 20, 2, 'PERCHA S1' FROM DUAL
        UNION ALL
        SELECT 'CASC-001', 12, 1, 'PERCHA S2' FROM DUAL
    ) datos
    INNER JOIN PRODUCTO producto
        ON producto.CODIGO = datos.CODIGO
) fuente
ON (
    destino.ID_PRODUCTO = fuente.ID_PRODUCTO
)
WHEN MATCHED THEN
    UPDATE SET
        destino.STOCK_ACTUAL = fuente.STOCK_ACTUAL,
        destino.STOCK_RESERVADO = fuente.STOCK_RESERVADO,
        destino.UBICACION = fuente.UBICACION,
        destino.FECHA_ULTIMO_MOVIMIENTO = CURRENT_TIMESTAMP,
        destino.FECHA_ACTUALIZACION = CURRENT_TIMESTAMP
WHEN NOT MATCHED THEN
    INSERT (
        ID_PRODUCTO,
        STOCK_ACTUAL,
        STOCK_RESERVADO,
        UBICACION,
        FECHA_ULTIMO_MOVIMIENTO
    )
    VALUES (
        fuente.ID_PRODUCTO,
        fuente.STOCK_ACTUAL,
        fuente.STOCK_RESERVADO,
        fuente.UBICACION,
        CURRENT_TIMESTAMP
    );

PROMPT ============================================================
PROMPT CONFIRMACION DE LA TRANSACCION
PROMPT ============================================================

COMMIT;

PROMPT ============================================================
PROMPT VERIFICACION DE DATOS
PROMPT ============================================================

COLUMN ENTIDAD FORMAT A20

SELECT 'ROL' AS ENTIDAD, COUNT(*) AS TOTAL FROM ROL
UNION ALL
SELECT 'CATEGORIA', COUNT(*) FROM CATEGORIA
UNION ALL
SELECT 'PROVEEDOR', COUNT(*) FROM PROVEEDOR
UNION ALL
SELECT 'CLIENTE', COUNT(*) FROM CLIENTE
UNION ALL
SELECT 'PRODUCTO', COUNT(*) FROM PRODUCTO
UNION ALL
SELECT 'INVENTARIO', COUNT(*) FROM INVENTARIO
ORDER BY ENTIDAD;

PROMPT ============================================================
PROMPT PRODUCTOS CON STOCK BAJO
PROMPT ============================================================

COLUMN CODIGO FORMAT A12
COLUMN PRODUCTO FORMAT A42
COLUMN ESTADO_STOCK FORMAT A15

SELECT
    producto.CODIGO,
    producto.NOMBRE AS PRODUCTO,
    inventario.STOCK_DISPONIBLE,
    producto.STOCK_MINIMO,
    CASE
        WHEN inventario.STOCK_DISPONIBLE <= producto.STOCK_MINIMO
            THEN 'STOCK BAJO'
        ELSE 'NORMAL'
    END AS ESTADO_STOCK
FROM PRODUCTO producto
INNER JOIN INVENTARIO inventario
    ON inventario.ID_PRODUCTO = producto.ID_PRODUCTO
ORDER BY producto.CODIGO;

PROMPT ============================================================
PROMPT DATOS INICIALES CARGADOS CORRECTAMENTE
PROMPT ============================================================

EXIT SUCCESS
