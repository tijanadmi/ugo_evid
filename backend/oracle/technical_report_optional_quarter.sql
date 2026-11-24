-- Run as TED after technical_report_templates.sql, before deploying this version.
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK
ALTER TABLE TED.UGO_TEH_IZVESTAJ MODIFY (BROJ_KVARTALA NULL);
COMMENT ON COLUMN TED.UGO_TEH_IZVESTAJ.BROJ_KVARTALA IS
 'NULL kada sablon ne koristi oznaku broj_kvartala; inace potvrdjeni broj kvartala.';
COMMIT;
