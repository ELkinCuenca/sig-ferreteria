ALTER SESSION SET CONTAINER = FREEPDB1;

SET SERVEROUTPUT ON
SET SQLBLANKLINES ON
SET FEEDBACK ON
SET VERIFY OFF

PROMPT
PROMPT ============================================================
PROMPT AMPLIACION DE ALERTA_STOCK
PROMPT ============================================================

DECLARE
    v_count NUMBER;
BEGIN
    SELECT COUNT(*)
    INTO v_count
    FROM ALL_TAB_COLUMNS
    WHERE OWNER = 'SIGEFER_APP'
      AND TABLE_NAME = 'ALERTA_STOCK'
      AND COLUMN_NAME = 'OBSERVACION_ATENCION';

    IF v_count = 0 THEN
        EXECUTE IMMEDIATE '
            ALTER TABLE SIGEFER_APP.ALERTA_STOCK
            ADD OBSERVACION_ATENCION VARCHAR2(500)
        ';

        DBMS_OUTPUT.PUT_LINE(
            'Columna OBSERVACION_ATENCION creada.'
        );
    END IF;

    SELECT COUNT(*)
    INTO v_count
    FROM ALL_TAB_COLUMNS
    WHERE OWNER = 'SIGEFER_APP'
      AND TABLE_NAME = 'ALERTA_STOCK'
      AND COLUMN_NAME = 'ID_USUARIO_ATENCION';

    IF v_count = 0 THEN
        EXECUTE IMMEDIATE '
            ALTER TABLE SIGEFER_APP.ALERTA_STOCK
            ADD ID_USUARIO_ATENCION NUMBER
        ';

        DBMS_OUTPUT.PUT_LINE(
            'Columna ID_USUARIO_ATENCION creada.'
        );
    END IF;

    SELECT COUNT(*)
    INTO v_count
    FROM ALL_TAB_COLUMNS
    WHERE OWNER = 'SIGEFER_APP'
      AND TABLE_NAME = 'ALERTA_STOCK'
      AND COLUMN_NAME = 'FECHA_ACTUALIZACION';

    IF v_count = 0 THEN
        EXECUTE IMMEDIATE '
            ALTER TABLE SIGEFER_APP.ALERTA_STOCK
            ADD FECHA_ACTUALIZACION TIMESTAMP(6)
                DEFAULT CURRENT_TIMESTAMP NOT NULL
        ';

        DBMS_OUTPUT.PUT_LINE(
            'Columna FECHA_ACTUALIZACION creada.'
        );
    END IF;
END;
/

PROMPT
PROMPT ============================================================
PROMPT CLAVE FORANEA DEL USUARIO RESPONSABLE
PROMPT ============================================================

DECLARE
    v_count NUMBER;
BEGIN
    SELECT COUNT(*)
    INTO v_count
    FROM ALL_CONSTRAINTS
    WHERE OWNER = 'SIGEFER_APP'
      AND TABLE_NAME = 'ALERTA_STOCK'
      AND CONSTRAINT_NAME =
          'FK_ALERTA_USUARIO_ATENCION';

    IF v_count = 0 THEN
        EXECUTE IMMEDIATE '
            ALTER TABLE SIGEFER_APP.ALERTA_STOCK
            ADD CONSTRAINT FK_ALERTA_USUARIO_ATENCION
            FOREIGN KEY (ID_USUARIO_ATENCION)
            REFERENCES SIGEFER_APP.USUARIO (
                ID_USUARIO
            )
        ';

        DBMS_OUTPUT.PUT_LINE(
            'FK_ALERTA_USUARIO_ATENCION creada.'
        );
    END IF;
END;
/

PROMPT
PROMPT ============================================================
PROMPT REGLA CONSISTENTE DE ESTADO Y FECHA
PROMPT ============================================================

DECLARE
    v_count NUMBER;
BEGIN
    SELECT COUNT(*)
    INTO v_count
    FROM ALL_CONSTRAINTS
    WHERE OWNER = 'SIGEFER_APP'
      AND TABLE_NAME = 'ALERTA_STOCK'
      AND CONSTRAINT_NAME = 'CK_ALERTA_FECHA';

    IF v_count > 0 THEN
        EXECUTE IMMEDIATE '
            ALTER TABLE SIGEFER_APP.ALERTA_STOCK
            DROP CONSTRAINT CK_ALERTA_FECHA
        ';
    END IF;

    SELECT COUNT(*)
    INTO v_count
    FROM ALL_CONSTRAINTS
    WHERE OWNER = 'SIGEFER_APP'
      AND TABLE_NAME = 'ALERTA_STOCK'
      AND CONSTRAINT_NAME =
          'CK_ALERTA_ESTADO_FECHA';

    IF v_count = 0 THEN
        EXECUTE IMMEDIATE q'[
            ALTER TABLE SIGEFER_APP.ALERTA_STOCK
            ADD CONSTRAINT CK_ALERTA_ESTADO_FECHA
            CHECK (
                (
                    ESTADO = 'PENDIENTE'
                    AND FECHA_ATENCION IS NULL
                )
                OR
                (
                    ESTADO IN (
                        'ATENDIDA',
                        'DESCARTADA'
                    )
                    AND FECHA_ATENCION IS NOT NULL
                )
            )
            ENABLE VALIDATE
        ]';

        DBMS_OUTPUT.PUT_LINE(
            'CK_ALERTA_ESTADO_FECHA creada.'
        );
    END IF;
END;
/

PROMPT
PROMPT ============================================================
PROMPT PROTECCION UNICODE DE LA NUEVA COLUMNA
PROMPT ============================================================

DECLARE
    v_count NUMBER;
BEGIN
    SELECT COUNT(*)
    INTO v_count
    FROM ALL_CONSTRAINTS
    WHERE OWNER = 'SIGEFER_APP'
      AND TABLE_NAME = 'ALERTA_STOCK'
      AND CONSTRAINT_NAME =
          'CKU_ALERTA_OBSERVACION';

    IF v_count = 0 THEN
        EXECUTE IMMEDIATE q'[
            ALTER TABLE SIGEFER_APP.ALERTA_STOCK
            ADD CONSTRAINT CKU_ALERTA_OBSERVACION
            CHECK (
                OBSERVACION_ATENCION IS NULL
                OR INSTR(
                    OBSERVACION_ATENCION,
                    UNISTR('\FFFD')
                ) = 0
            )
            ENABLE VALIDATE
        ]';

        DBMS_OUTPUT.PUT_LINE(
            'CKU_ALERTA_OBSERVACION creada.'
        );
    END IF;
END;
/

PROMPT
PROMPT ============================================================
PROMPT INDICES
PROMPT ============================================================

DECLARE
    v_count NUMBER;
BEGIN
    SELECT COUNT(*)
    INTO v_count
    FROM ALL_INDEXES
    WHERE OWNER = 'SIGEFER_APP'
      AND INDEX_NAME =
          'IX_ALERTA_USUARIO_ATENCION';

    IF v_count = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE INDEX
                SIGEFER_APP.IX_ALERTA_USUARIO_ATENCION
            ON SIGEFER_APP.ALERTA_STOCK (
                ID_USUARIO_ATENCION
            )
        ';

        DBMS_OUTPUT.PUT_LINE(
            'IX_ALERTA_USUARIO_ATENCION creado.'
        );
    END IF;
END;
/

PROMPT
PROMPT ============================================================
PROMPT VERIFICACION
PROMPT ============================================================

COLUMN COLUMN_NAME FORMAT A30
COLUMN DATA_TYPE FORMAT A22
COLUMN NULLABLE FORMAT A10
COLUMN CONSTRAINT_NAME FORMAT A35
COLUMN STATUS FORMAT A10

SELECT
    COLUMN_ID,
    COLUMN_NAME,
    DATA_TYPE,
    NULLABLE
FROM ALL_TAB_COLUMNS
WHERE OWNER = 'SIGEFER_APP'
  AND TABLE_NAME = 'ALERTA_STOCK'
ORDER BY COLUMN_ID;

SELECT
    CONSTRAINT_NAME,
    CONSTRAINT_TYPE,
    STATUS
FROM ALL_CONSTRAINTS
WHERE OWNER = 'SIGEFER_APP'
  AND TABLE_NAME = 'ALERTA_STOCK'
ORDER BY CONSTRAINT_NAME;

EXIT;
