-- Run once as TED before deploying the new backend.
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK

DECLARE
 n NUMBER;
BEGIN
 SELECT COUNT(*) INTO n FROM user_constraints fk JOIN user_constraints pk
 ON pk.constraint_name=fk.r_constraint_name
 WHERE fk.constraint_type='R' AND pk.table_name='UGO_DOC';
 IF n>0 THEN
  raise_application_error(-20001,'UGO_DOC still has incoming foreign keys. Complete the legacy document-link cleanup before enabling physical deletion.');
 END IF;
END;
/

-- Durable deletion intent and audit. No FK to UGO_DOC: the audit survives it.
CREATE TABLE TED.UGO_DOC_BRISANJE (
 ID_DOC NUMBER(19,0) NOT NULL,
 ID_UGO_EVID NUMBER(19,0) NOT NULL,
 ORIGINALNI_NAZIV VARCHAR2(255 CHAR) NOT NULL,
 ID_KORISNIKA NUMBER(19,0) NOT NULL,
 TRAZENO TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
 ZAVRSENO TIMESTAMP,
 CONSTRAINT PK_UGO_DOC_BRISANJE PRIMARY KEY(ID_DOC)
);
CREATE INDEX TED.IX_DOC_BRISANJE_PENDING ON TED.UGO_DOC_BRISANJE (ZAVRSENO,TRAZENO);
COMMENT ON COLUMN TED.UGO_DOC.STATUS IS 'P: pending upload; A: available; D: pending physical deletion when UGO_DOC_BRISANJE entry exists, otherwise historical soft deletion';
COMMENT ON TABLE TED.UGO_DOC_BRISANJE IS 'Deletion request and audit; retained after UGO_DOC and file removal. Historical D rows are not automatically queued.';
COMMIT;
