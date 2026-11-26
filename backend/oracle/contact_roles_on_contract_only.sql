-- Run as TED AFTER contract_people.sql, with the old backend stopped.
-- Back up UGO_DOB_LICA first. Deploy the matching backend and frontend together.
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK

-- Review external views/triggers/packages that may still use the removed column.
SELECT OWNER, NAME, TYPE FROM ALL_DEPENDENCIES
 WHERE REFERENCED_OWNER='TED' AND REFERENCED_NAME='UGO_DOB_LICA'
 ORDER BY OWNER, TYPE, NAME;

DECLARE
 n NUMBER;
 missing_links NUMBER;
BEGIN
 SELECT COUNT(*) INTO n FROM ALL_TABLES
 WHERE OWNER='TED' AND TABLE_NAME='UGO_EVID_LICA';
 IF n<>1 THEN
  RAISE_APPLICATION_ERROR(-20001,'Run contract_people.sql before removing the directory role.');
 END IF;

 -- Existing explicit SLM links must already be migrated, including historical ones.
 EXECUTE IMMEDIATE 'SELECT COUNT(*) FROM TED.UGO_EVID e
 WHERE e.id_ugo_dob_lica IS NOT NULL AND NOT EXISTS
 (SELECT 1 FROM TED.UGO_EVID_LICA a WHERE a.id_ugo_evid=e.id
 AND a.id_ugo_dob_lica=e.id_ugo_dob_lica AND a.id_ugo_dob_lica_rola=1)'
 INTO missing_links;
 IF missing_links>0 THEN
  RAISE_APPLICATION_ERROR(-20002,'Some legacy SLM links have not been migrated. Review them first.');
 END IF;

 SELECT COUNT(*) INTO n FROM ALL_TAB_COLUMNS
 WHERE OWNER='TED' AND TABLE_NAME='UGO_DOB_LICA' AND COLUMN_NAME='ID_UGO_DOB_LICA_ROLA';
 IF n=1 THEN
  EXECUTE IMMEDIATE 'ALTER TABLE TED.UGO_DOB_LICA DROP COLUMN ID_UGO_DOB_LICA_ROLA';
 END IF;
END;
/
-- UGO_DOB_LICA_ROLE and UGO_EVID_LICA.ID_UGO_DOB_LICA_ROLA are retained.
