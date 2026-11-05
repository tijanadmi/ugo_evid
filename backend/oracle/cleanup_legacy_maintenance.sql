-- Run ONCE as TED, in SQL*Plus/SQLcl or SQL Developer Run Script (F5).
-- STOP ALL application instances first. Deploy compatible backend before restart.
-- Requires the current schema, including both UGO_DOC tracking columns.
-- This script changes data and drops FIVE legacy columns. DDL auto-commits.
-- The backup tables are intentional: do not delete them until migration is verified.
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK
SET DEFINE OFF
SET SQLBLANKLINES ON

-- Preflight: a document can belong to only one request OR one period.
-- Reject ambiguous/shared legacy links rather than silently losing associations.
DECLARE
 n NUMBER;
BEGIN
 IF USER <> 'TED' THEN
  RAISE_APPLICATION_ERROR(-20001, 'Connect as TED.');
 END IF;
 SELECT COUNT(*) INTO n FROM user_objects WHERE object_name IN
  ('UGO_BAK_RAZ_LEGACY','UGO_BAK_TEH_LEGACY','UGO_BAK_ODR_LEGACY');
 IF n>0 THEN
  RAISE_APPLICATION_ERROR(-20002, 'Backup objects already exist. Inspect previous migration before continuing.');
 END IF;
 FOR x IN (
  SELECT doc_id FROM (
   SELECT id_zahtev_doc doc_id, 'R' kind, id owner_id FROM TED.UGO_RAZ_ZAHTEV WHERE id_zahtev_doc IS NOT NULL
   UNION
   SELECT id_ugo_doc, 'R', id FROM TED.UGO_RAZ_ZAHTEV WHERE id_ugo_doc IS NOT NULL
   UNION
   SELECT id_ugo_doc, 'T', id FROM TED.UGO_TEH_PERIOD WHERE id_ugo_doc IS NOT NULL
   UNION
   SELECT id, 'R', id_raz_zahtev FROM TED.UGO_DOC WHERE id_raz_zahtev IS NOT NULL
   UNION
   SELECT id, 'T', id_teh_period FROM TED.UGO_DOC WHERE id_teh_period IS NOT NULL
  ) GROUP BY doc_id HAVING COUNT(*)>1
 ) LOOP
  RAISE_APPLICATION_ERROR(-20003, 'Document has multiple owners; resolve manually. UGO_DOC.ID='||x.doc_id);
 END LOOP;
 FOR x IN (
  SELECT z.id_zahtev_doc doc_id, o.id_ugo_evid evid_id
   FROM TED.UGO_RAZ_ZAHTEV z JOIN TED.UGO_RAZ_FOND f ON f.id=z.id_raz_fonda
   JOIN TED.UGO_ODRZAVANJE o ON o.id=f.id_odrzavanja WHERE z.id_zahtev_doc IS NOT NULL
  UNION ALL
  SELECT z.id_ugo_doc, o.id_ugo_evid
   FROM TED.UGO_RAZ_ZAHTEV z JOIN TED.UGO_RAZ_FOND f ON f.id=z.id_raz_fonda
   JOIN TED.UGO_ODRZAVANJE o ON o.id=f.id_odrzavanja WHERE z.id_ugo_doc IS NOT NULL
  UNION ALL
  SELECT p.id_ugo_doc, o.id_ugo_evid
   FROM TED.UGO_TEH_PERIOD p JOIN TED.UGO_ODRZAVANJE o ON o.id=p.id_odrzavanja
   WHERE p.id_ugo_doc IS NOT NULL
 ) LOOP
  SELECT COUNT(*) INTO n FROM TED.UGO_DOC WHERE id=x.doc_id AND id_ugo_evid=x.evid_id;
  IF n<>1 THEN
   RAISE_APPLICATION_ERROR(-20004, 'Missing document or different contract. UGO_DOC.ID='||x.doc_id);
  END IF;
 END LOOP;
 -- Check orphan parent references too, even if old FKs were removed by CASCADE.
 SELECT COUNT(*) INTO n FROM TED.UGO_RAZ_ZAHTEV z
  WHERE NOT EXISTS (SELECT 1 FROM TED.UGO_RAZ_FOND f JOIN TED.UGO_ODRZAVANJE o ON o.id=f.id_odrzavanja WHERE f.id=z.id_raz_fonda);
 IF n>0 THEN RAISE_APPLICATION_ERROR(-20005, 'Requests with missing fund/maintenance. Repair parent links first.'); END IF;
 SELECT COUNT(*) INTO n FROM TED.UGO_TEH_PERIOD p
  WHERE NOT EXISTS (SELECT 1 FROM TED.UGO_ODRZAVANJE o WHERE o.id=p.id_odrzavanja);
 IF n>0 THEN RAISE_APPLICATION_ERROR(-20006, 'Periods with missing maintenance. Repair parent links first.'); END IF;
END;
/

-- Preserve original rows, including old dates, for review/recovery.
-- These are data snapshots, not full database backups (no copied constraints).
CREATE TABLE TED.UGO_BAK_RAZ_LEGACY AS SELECT * FROM TED.UGO_RAZ_ZAHTEV;
CREATE TABLE TED.UGO_BAK_TEH_LEGACY AS SELECT * FROM TED.UGO_TEH_PERIOD;
CREATE TABLE TED.UGO_BAK_ODR_LEGACY AS SELECT * FROM TED.UGO_ODRZAVANJE;

MERGE INTO TED.UGO_DOC d USING (
 SELECT id_zahtev_doc doc_id, id owner_id FROM TED.UGO_RAZ_ZAHTEV WHERE id_zahtev_doc IS NOT NULL
 UNION
 SELECT id_ugo_doc, id FROM TED.UGO_RAZ_ZAHTEV WHERE id_ugo_doc IS NOT NULL
) s ON (d.id=s.doc_id)
WHEN MATCHED THEN UPDATE SET d.id_raz_zahtev=s.owner_id, d.version=d.version+1
 WHERE d.id_raz_zahtev IS NULL;

MERGE INTO TED.UGO_DOC d USING (
 SELECT id_ugo_doc doc_id, id owner_id FROM TED.UGO_TEH_PERIOD WHERE id_ugo_doc IS NOT NULL
) s ON (d.id=s.doc_id)
WHEN MATCHED THEN UPDATE SET d.id_teh_period=s.owner_id, d.version=d.version+1
 WHERE d.id_teh_period IS NULL;

-- Verify every original association before any column is dropped.
DECLARE
 n NUMBER;
BEGIN
 SELECT COUNT(*) INTO n FROM (
  SELECT id_zahtev_doc doc_id, id owner_id FROM TED.UGO_BAK_RAZ_LEGACY WHERE id_zahtev_doc IS NOT NULL
  UNION
  SELECT id_ugo_doc, id FROM TED.UGO_BAK_RAZ_LEGACY WHERE id_ugo_doc IS NOT NULL
 ) s WHERE NOT EXISTS (SELECT 1 FROM TED.UGO_DOC d WHERE d.id=s.doc_id AND d.id_raz_zahtev=s.owner_id);
 IF n>0 THEN RAISE_APPLICATION_ERROR(-20007, 'Request document migration incomplete.'); END IF;
 SELECT COUNT(*) INTO n FROM TED.UGO_BAK_TEH_LEGACY p WHERE p.id_ugo_doc IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM TED.UGO_DOC d WHERE d.id=p.id_ugo_doc AND d.id_teh_period=p.id);
 IF n>0 THEN RAISE_APPLICATION_ERROR(-20008, 'Period document migration incomplete.'); END IF;
END;
/
COMMIT;

-- Remove only constraints involving the removed columns, including unnamed FKs.
-- Preserve UGO_DOC -> request/period FKs and all other business relations.
ALTER TABLE TED.UGO_RAZ_ZAHTEV DROP (ID_ZAHTEV_DOC, ID_UGO_DOC) CASCADE CONSTRAINTS;
ALTER TABLE TED.UGO_TEH_PERIOD DROP COLUMN ID_UGO_DOC CASCADE CONSTRAINTS;
ALTER TABLE TED.UGO_ODRZAVANJE DROP (DATUM_POTPISA, DATUM_KRAJA) CASCADE CONSTRAINTS;

-- POCETAK_ODRZAVANJA remains an explicit optional override, never inferred
-- from the old signature date. Contract dates are read from SAP_UGOVORI.
SELECT table_name, column_name FROM user_tab_columns
 WHERE (table_name='UGO_RAZ_ZAHTEV' AND column_name IN ('ID_ZAHTEV_DOC','ID_UGO_DOC'))
 OR (table_name='UGO_TEH_PERIOD' AND column_name='ID_UGO_DOC')
 OR (table_name='UGO_ODRZAVANJE' AND column_name IN ('DATUM_POTPISA','DATUM_KRAJA'));
-- Expected: no rows.
