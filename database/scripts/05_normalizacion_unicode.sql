ALTER SESSION SET CONTAINER = FREEPDB1;

SET SERVEROUTPUT ON
SET PAGESIZE 100
SET LINESIZE 220
SET FEEDBACK ON

PROMPT
PROMPT === CORRECCION DE TEXTOS CON UNICODE ===

UPDATE SIGEFER_APP.PRODUCTO
SET NOMBRE = UNISTR(
    'Cable flexible n\00FAmero 12'
)
WHERE CODIGO = 'CABL-001';

COMMIT;

PROMPT
PROMPT === VERIFICACION DEL PRODUCTO ===

COLUMN CODIGO FORMAT A15
COLUMN TEXTO_NORMALIZADO FORMAT A100

SELECT
    CODIGO,
    ASCIISTR(NOMBRE) AS TEXTO_NORMALIZADO
FROM SIGEFER_APP.PRODUCTO
WHERE CODIGO = 'CABL-001';

PROMPT
PROMPT === AUDITORIA GLOBAL DE CARACTERES SOSPECHOSOS ===

DECLARE
    v_sql        VARCHAR2(32767);
    v_count      NUMBER;
    v_total      NUMBER := 0;
BEGIN
    FOR column_record IN (
        SELECT
            TABLE_NAME,
            COLUMN_NAME
        FROM ALL_TAB_COLUMNS
        WHERE OWNER = 'SIGEFER_APP'
          AND DATA_TYPE IN (
              'CHAR',
              'NCHAR',
              'VARCHAR2',
              'NVARCHAR2'
          )
        ORDER BY
            TABLE_NAME,
            COLUMN_ID
    ) LOOP
        v_sql :=
            'SELECT COUNT(*) ' ||
            'FROM SIGEFER_APP.' ||
            column_record.TABLE_NAME ||
            ' WHERE ' ||
            'INSTR(' ||
            column_record.COLUMN_NAME ||
            ', UNISTR(''\FFFD'')) > 0 ' ||
            'OR INSTR(' ||
            column_record.COLUMN_NAME ||
            ', UNISTR(''\00C3'')) > 0 ' ||
            'OR INSTR(' ||
            column_record.COLUMN_NAME ||
            ', UNISTR(''\00C2'')) > 0';

        EXECUTE IMMEDIATE v_sql
        INTO v_count;

        IF v_count > 0 THEN
            DBMS_OUTPUT.PUT_LINE(
                column_record.TABLE_NAME ||
                '.' ||
                column_record.COLUMN_NAME ||
                ': ' ||
                v_count ||
                ' registros sospechosos'
            );

            v_total := v_total + v_count;
        END IF;
    END LOOP;

    IF v_total = 0 THEN
        DBMS_OUTPUT.PUT_LINE(
            'OK: no se encontraron caracteres sospechosos.'
        );
    ELSE
        DBMS_OUTPUT.PUT_LINE(
            'TOTAL DE COINCIDENCIAS: ' ||
            v_total
        );
    END IF;
END;
/

EXIT;
