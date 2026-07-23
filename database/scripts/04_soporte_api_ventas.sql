SET ECHO ON
SET SQLBLANKLINES ON
SET FEEDBACK ON
SET SERVEROUTPUT ON
SET DEFINE OFF

WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK
WHENEVER OSERROR EXIT FAILURE

PROMPT ============================================================
PROMPT SIGEFER - SOPORTE PARA TRANSACCIONES DE VENTA
PROMPT ============================================================

DECLARE
    v_total NUMBER;
BEGIN
    SELECT COUNT(*)
    INTO v_total
    FROM USER_SEQUENCES
    WHERE SEQUENCE_NAME = 'SEQ_NUMERO_VENTA';

    IF v_total = 0 THEN
        EXECUTE IMMEDIATE '
            CREATE SEQUENCE SEQ_NUMERO_VENTA
            START WITH 1
            INCREMENT BY 1
            NOCYCLE
            CACHE 20
        ';

        DBMS_OUTPUT.PUT_LINE(
            'Secuencia SEQ_NUMERO_VENTA creada'
        );
    ELSE
        DBMS_OUTPUT.PUT_LINE(
            'Secuencia SEQ_NUMERO_VENTA ya existe'
        );
    END IF;
END;
/

COLUMN SEQUENCE_NAME FORMAT A30

SELECT
    SEQUENCE_NAME,
    INCREMENT_BY,
    CACHE_SIZE,
    CYCLE_FLAG
FROM USER_SEQUENCES
WHERE SEQUENCE_NAME = 'SEQ_NUMERO_VENTA';

PROMPT ============================================================
PROMPT SOPORTE PARA VENTAS CONFIGURADO CORRECTAMENTE
PROMPT ============================================================

EXIT SUCCESS
