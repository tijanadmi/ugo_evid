-- Run ONCE as TED on the existing database, after development_request_documents.sql.
-- Stop the application during migration. Oracle DDL commits automatically.
-- ROK_POTPISA keeps its physical name for compatibility, but means completion deadline.
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK

-- Add and validate replacement rules BEFORE removing the old rule.
-- If existing records violate a rule, stop and review them; do not change dates blindly.
ALTER TABLE TED.UGO_RAZ_ZAHTEV ADD CONSTRAINT CK_RAZ_REAL_ROK
 CHECK (DATUM_REALIZACIJE IS NULL OR DATUM_REALIZACIJE <= ROK_POTPISA)
 ENABLE VALIDATE;
ALTER TABLE TED.UGO_RAZ_ZAHTEV ADD CONSTRAINT CK_RAZ_POTPIS_OD
 CHECK (DATUM_POTPISA IS NULL OR DATUM_POTPISA >= DATUM_ZAHTEVA)
 ENABLE VALIDATE;
ALTER TABLE TED.UGO_RAZ_ZAHTEV DROP CONSTRAINT CK_RAZ_ROK_POTPISA;

-- CK_RAZ_REALIZACIJA still ensures completion is not after signature.
COMMENT ON COLUMN TED.UGO_RAZ_ZAHTEV.ROK_POTPISA IS
 'Service completion deadline, inclusive; counted from the day after request using the approved work calendar. Signature may follow this date.';
