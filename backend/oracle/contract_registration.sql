-- Run with SQL*Plus/SQLcl as TED (or an account allowed to alter TED objects).
-- No data is deleted or corrected. Existing duplicates must be resolved first.
WHENEVER SQLERROR EXIT SQL.SQLCODE

SELECT id_sap_ugovor, COUNT(*) AS broj
FROM TED.UGO_EVID
GROUP BY id_sap_ugovor HAVING COUNT(*) > 1;

SELECT id_sap_dobavljac, COUNT(*) AS broj
FROM TED.UGO_DOB_LICA
WHERE status = 'A' AND id_ugo_dob_lica_rola = 1
GROUP BY id_sap_dobavljac HAVING COUNT(*) > 1;

-- Stop before making any schema changes if either invariant is violated.
DECLARE
 duplicates NUMBER;
BEGIN
 SELECT COUNT(*) INTO duplicates FROM (
  SELECT id_sap_ugovor FROM TED.UGO_EVID GROUP BY id_sap_ugovor HAVING COUNT(*) > 1
 );
 IF duplicates > 0 THEN
  RAISE_APPLICATION_ERROR(-20001, 'UGO_EVID contains duplicate SAP contracts; review them first.');
 END IF;
 SELECT COUNT(*) INTO duplicates FROM (
  SELECT id_sap_dobavljac FROM TED.UGO_DOB_LICA
  WHERE status = 'A' AND id_ugo_dob_lica_rola = 1
  GROUP BY id_sap_dobavljac HAVING COUNT(*) > 1
 );
 IF duplicates > 0 THEN
  RAISE_APPLICATION_ERROR(-20002, 'Multiple active role-1 contacts exist; review them first.');
 END IF;
END;
/

-- Required database-wide invariant; its unique index also supports NOT EXISTS.
ALTER TABLE TED.UGO_EVID ADD CONSTRAINT UQ_UGO_EVID_SAP UNIQUE (ID_SAP_UGOVOR);

-- At most one active role-1 contact per supplier. Inactive/other roles unrestricted.
CREATE UNIQUE INDEX TED.UQ_UGO_DOB_LICA_SLM_A ON TED.UGO_DOB_LICA (
 CASE WHEN STATUS = 'A' AND ID_UGO_DOB_LICA_ROLA = 1 THEN ID_SAP_DOBAVLJAC END
);

-- Recommended now for future editing; registration works without this column.
-- Run separately if VERSION does not already exist:
-- ALTER TABLE TED.UGO_EVID ADD VERSION NUMBER(19,0) DEFAULT 1 NOT NULL;
