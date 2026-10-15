-- Run ONCE as TED after maintenance_tracking.sql, with application stopped.
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK
ALTER TABLE TED.UGO_RADNI_KALENDAR ADD (
 IZVOR VARCHAR2(12 CHAR) DEFAULT 'GENERISAN' NOT NULL,
 CONSTRAINT CK_RADNI_IZVOR CHECK (IZVOR IN ('GENERISAN','RUCNO','PRAZNIK'))
);
-- Legacy rows have no audit of manual edits. Protect approved days, descriptions
-- and deviations from the weekday/weekend default conservatively.
UPDATE TED.UGO_RADNI_KALENDAR SET IZVOR='RUCNO'
WHERE POTVRDJEN='D' OR (OPIS IS NOT NULL AND OPIS<>'Predlog - proveriti praznike')
OR RADNI<>CASE WHEN DATUM-TRUNC(DATUM,'IW') IN (5,6) THEN 'N' ELSE 'D' END;
COMMENT ON COLUMN TED.UGO_RADNI_KALENDAR.IZVOR IS 'GENERISAN default calendar; RUCNO protected manual correction; PRAZNIK holiday import';
COMMIT;
