# Technical support reports: download only

Reports are rendered from the current UGO_TEH_PERIOD and current contract template.
Select DOCX or PDF, then Generate and download. No report history or permanent
report files are created. DOCX stays in memory; PDF uses a private temporary
folder removed on return, including converter errors. An abrupt OS/process kill
can leave temporary files in the system temp directory.

## Database setup

Run scripts as TED, only for objects not already created:

1. `technical_period_quarter.sql` adds UGO_TEH_PERIOD.BROJ_KVARTALA.
   Skip it if that column already exists.
2. `technical_report_templates.sql` creates only UGO_DOC_TEMPLATE.
   Run it once if that table does not yet exist.
3. Deploy the backend and frontend together.

No report history table or deletion migration is needed. Existing uploaded
UGO_DOC documents remain unchanged. These scripts are not executed automatically.

## Templates and values

Templates remain stored per contract/document type with immutable versions.
Keep fixed text, RUN number and signatory in the Word template.
Required markers: {{datum_od}}, {{datum_do}}, {{datum_potpisa}}.
Legacy {{sysdate}} means the stored signature date, NOT today's date.
Dates are formatted dd.mm.yyyy.

When {{broj_kvartala}} is present, the dialog offers a quarter and saves the
confirmed value in UGO_TEH_PERIOD. Otherwise it hides that field and preserves
any existing period quarter. Generation does not change the period status.
The signature date must already exist in the period.
Signed final documents can still be uploaded through the existing document UI.
Regeneration uses current data/template, not an archived historical snapshot.

## PDF setup

Install LibreOffice Writer and template fonts on the backend machine.
Set LIBREOFFICE_PATH in private app.env, e.g. /usr/bin/libreoffice on Ubuntu or
C:/Program Files/LibreOffice/program/soffice.com on Windows.
Without this setting only DOCX is selectable. Conversion has a time limit and
an isolated LibreOffice profile for each request.

POST /ugo_evid/:id/teh-period/:period/izvestaj accepts id_template, version
(string), format (docx/pdf), and optional broj_kvartala. It returns attachment
bytes with Cache-Control: private, no-store. Old report history routes are removed.
Contract write permission, lease and snapshot/version checks still apply.
