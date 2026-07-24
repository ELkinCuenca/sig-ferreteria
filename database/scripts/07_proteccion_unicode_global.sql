ALTER SESSION SET CONTAINER = FREEPDB1;

SET SERVEROUTPUT ON
SET FEEDBACK ON
SET VERIFY OFF

PROMPT
PROMPT ============================================================
PROMPT PROTECCION GLOBAL CONTRA U+FFFD
PROMPT ============================================================

DECLARE
    v_sql             VARCHAR2(32767);
    v_table           VARCHAR2(300);
    v_column          VARCHAR2(300);
    v_constraint      VARCHAR2(128);
    v_hash            VARCHAR2(6);
    v_exists          NUMBER;
    v_created         NUMBER := 0;
BEGIN
    FOR column_record IN (
        SELECT
            columns.TABLE_NAME,
            columns.COLUMN_NAME
        FROM ALL_TAB_COLUMNS columns
        WHERE columns.OWNER = 'SIGEFER_APP'
          AND columns.DATA_TYPE IN (
              'CHAR',
              'VARCHAR2',
              'NCHAR',
              'NVARCHAR2'
          )
          AND columns.TABLE_NAME <> 'UNICODE_HALLAZGO'
          AND EXISTS (
              SELECT 1
              FROM ALL_TABLES tables
              WHERE tables.OWNER = columns.OWNER
                AND tables.TABLE_NAME =
                    columns.TABLE_NAME
          )
        ORDER BY
            columns.TABLE_NAME,
            columns.COLUMN_ID
    ) LOOP
        v_table := DBMS_ASSERT.ENQUOTE_NAME(
            column_record.TABLE_NAME,
            FALSE
        );

        v_column := DBMS_ASSERT.ENQUOTE_NAME(
            column_record.COLUMN_NAME,
            FALSE
        );

        v_hash := LPAD(
            TO_CHAR(
                DBMS_UTILITY.GET_HASH_VALUE(
                    column_record.TABLE_NAME ||
                    ':' ||
                    column_record.COLUMN_NAME,
                    0,
                    999999
                )
            ),
            6,
            '0'
        );

        v_constraint :=
            'CKU_' ||
            SUBSTR(column_record.TABLE_NAME, 1, 8) ||
            '_' ||
            SUBSTR(column_record.COLUMN_NAME, 1, 8) ||
            '_' ||
            v_hash;

        SELECT
            COUNT(*)
        INTO v_exists
        FROM ALL_CONSTRAINTS
        WHERE OWNER = 'SIGEFER_APP'
          AND CONSTRAINT_NAME = v_constraint;

        IF v_exists = 0 THEN
            v_sql :=
                'ALTER TABLE SIGEFER_APP.' ||
                v_table ||
                ' ADD CONSTRAINT ' ||
                DBMS_ASSERT.SIMPLE_SQL_NAME(
                    v_constraint
                ) ||
                ' CHECK (' ||
                v_column ||
                ' IS NULL OR INSTR(' ||
                v_column ||
                ', UNISTR(''\FFFD'')) = 0' ||
                ') ENABLE VALIDATE';

            EXECUTE IMMEDIATE v_sql;

            v_created := v_created + 1;

            DBMS_OUTPUT.PUT_LINE(
                'Creada: ' ||
                v_constraint ||
                ' en ' ||
                column_record.TABLE_NAME ||
                '.' ||
                column_record.COLUMN_NAME
            );
        END IF;
    END LOOP;

    DBMS_OUTPUT.PUT_LINE(
        'Restricciones nuevas: ' ||
        v_created
    );
END;
/

PROMPT
PROMPT ============================================================
PROMPT RESTRICCIONES UNICODE ACTIVAS
PROMPT ============================================================

SELECT
    TABLE_NAME,
    CONSTRAINT_NAME,
    STATUS,
    VALIDATED
FROM ALL_CONSTRAINTS
WHERE OWNER = 'SIGEFER_APP'
  AND CONSTRAINT_NAME LIKE 'CKU\_%' ESCAPE '\'
ORDER BY
    TABLE_NAME,
    CONSTRAINT_NAME;

EXIT;
