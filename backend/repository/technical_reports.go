package repository

import (
	"context"
	"database/sql"
	"github.com/tijanadmi/ugo_evid/models"
)

type TechnicalReportStore interface {
	ListReportTemplates(context.Context, int) ([]models.ReportTemplate, error)
	SaveReportTemplate(context.Context, *models.ReportTemplate, int, int) error
	ListTechnicalReports(context.Context, int, int) ([]models.TechnicalReport, error)
	SaveTechnicalReport(context.Context, *models.TechnicalReport, int64, int) error
}

// Serialize template changes and report generation with maintenance edits.
func reportTransaction(ctx context.Context, db *sql.DB, contract int) (*sql.Tx, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	fail := func(e error) (*sql.Tx, error) { tx.Rollback(); return nil, e }
	if err = openDocumentContract(ctx, tx, contract); err != nil {
		return fail(err)
	}
	var id int
	if err = tx.QueryRowContext(ctx, `SELECT id FROM TED.UGO_ODRZAVANJE WHERE id_ugo_evid=:id FOR UPDATE NOWAIT`, sql.Named("id", contract)).Scan(&id); err != nil {
		return fail(err)
	}
	var locks int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE entity_type='UGO_ODRZAVANJE' AND entity_id=:id AND expires_at>SYSTIMESTAMP`, sql.Named("id", contract)).Scan(&locks); err != nil {
		return fail(err)
	}
	if locks > 0 {
		return fail(ErrDocumentLocked)
	}
	return tx, nil
}
func (r *OracleStore) ListReportTemplates(ctx context.Context, contract int) ([]models.ReportTemplate, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,id_ugo_evid,id_tip_doc,verzija,originalni_naziv,putanja FROM TED.UGO_DOC_TEMPLATE WHERE id_ugo_evid=:id ORDER BY id_tip_doc,verzija DESC`, sql.Named("id", contract))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ReportTemplate{}
	for rows.Next() {
		var t models.ReportTemplate
		if err = rows.Scan(&t.ID, &t.ContractID, &t.TypeID, &t.Version, &t.Name, &t.Path); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (r *OracleStore) SaveReportTemplate(ctx context.Context, t *models.ReportTemplate, expected, user int) error {
	tx, err := reportTransaction(ctx, r.DB, t.ContractID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var valid int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_TIP_DOC WHERE id=:id AND status='A' AND sifra IN ('POTVRDA_TEH','ZAPISNIK_TEH')`, sql.Named("id", t.TypeID)).Scan(&valid); err != nil {
		return err
	}
	if valid != 1 {
		return ErrDocumentType
	}
	var current int
	if err = tx.QueryRowContext(ctx, `SELECT NVL(MAX(verzija),0) FROM TED.UGO_DOC_TEMPLATE WHERE id_ugo_evid=:id AND id_tip_doc=:dtype`, sql.Named("id", t.ContractID), sql.Named("dtype", t.TypeID)).Scan(&current); err != nil {
		return err
	}
	if current != expected {
		return ErrDocumentVersion
	}
	t.Version = current + 1
	_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_DOC_TEMPLATE(id_ugo_evid,id_tip_doc,verzija,originalni_naziv,putanja,id_korisnika) VALUES(:contract,:dtype,:version,:name,:path,:actor) RETURNING id INTO :out_id`, sql.Named("contract", t.ContractID), sql.Named("dtype", t.TypeID), sql.Named("version", t.Version), sql.Named("name", t.Name), sql.Named("path", t.Path), sql.Named("actor", user), sql.Named("out_id", sql.Out{Dest: &t.ID}))
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *OracleStore) ListTechnicalReports(ctx context.Context, contract, period int) ([]models.TechnicalReport, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,id_ugo_evid,id_teh_period,id_template,broj_kvartala,TO_CHAR(period_od,'YYYY-MM-DD'),TO_CHAR(period_do,'YYYY-MM-DD'),TO_CHAR(datum_potpisa,'YYYY-MM-DD'),docx_putanja,pdf_putanja,TO_CHAR(datpri,'DD.MM.YYYY HH24:MI') FROM TED.UGO_TEH_IZVESTAJ WHERE id_ugo_evid=:contract AND id_teh_period=:period ORDER BY id DESC`, sql.Named("contract", contract), sql.Named("period", period))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.TechnicalReport{}
	for rows.Next() {
		var p models.TechnicalReport
		if err = scanNullable(rows, &p.ID, &p.ContractID, &p.PeriodID, &p.TemplateID, &p.Quarter, &p.Start, &p.End, &p.Signed, &p.DOCX, &p.PDF, &p.Created); err != nil {
			return nil, err
		}
		p.HasPDF = p.PDF != ""
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *OracleStore) SaveTechnicalReport(ctx context.Context, p *models.TechnicalReport, version int64, user int) error {
	if p.Quarter < 0 || p.Quarter > 9999 || p.Signed == "" {
		return ErrTrackingInput
	}
	tx, err := reportTransaction(ctx, r.DB, p.ContractID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current int64
	var dtype int
	var start, end, signed, status string
	err = scanNullable(tx.QueryRowContext(ctx, `SELECT p.version,p.id_tip_doc,TO_CHAR(p.period_od,'YYYY-MM-DD'),TO_CHAR(p.period_do,'YYYY-MM-DD'),TO_CHAR(p.datum_potpisa,'YYYY-MM-DD'),p.status FROM TED.UGO_TEH_PERIOD p JOIN TED.UGO_ODRZAVANJE o ON o.id=p.id_odrzavanja WHERE p.id=:period AND o.id_ugo_evid=:contract FOR UPDATE OF p.version NOWAIT`, sql.Named("period", p.PeriodID), sql.Named("contract", p.ContractID)), &current, &dtype, &start, &end, &signed, &status)
	if err != nil {
		return err
	}
	if version != current || start != p.Start || end != p.End || signed != p.Signed || status == "OTKAZAN" {
		return ErrMaintenanceConflict
	}
	var template int
	err = tx.QueryRowContext(ctx, `SELECT id FROM TED.UGO_DOC_TEMPLATE WHERE id_ugo_evid=:contract AND id_tip_doc=:dtype ORDER BY verzija DESC FETCH FIRST 1 ROWS ONLY`, sql.Named("contract", p.ContractID), sql.Named("dtype", dtype)).Scan(&template)
	if err != nil {
		return err
	}
	if template != p.TemplateID {
		return ErrDocumentVersion
	}
	_, err = tx.ExecContext(ctx, `UPDATE TED.UGO_TEH_PERIOD SET broj_kvartala=CASE WHEN :quarter=0 THEN broj_kvartala ELSE :quarter_value END,version=version+1,datizm=SYSTIMESTAMP,id_korisnika=:actor WHERE id=:id`, sql.Named("quarter", p.Quarter), sql.Named("quarter_value", p.Quarter), sql.Named("actor", user), sql.Named("id", p.PeriodID))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_TEH_IZVESTAJ(id_ugo_evid,id_teh_period,id_template,broj_kvartala,period_od,period_do,datum_potpisa,docx_putanja,pdf_putanja,id_korisnika) VALUES(:contract,:period,:template,NULLIF(:quarter,0),TO_DATE(:start_date,'YYYY-MM-DD'),TO_DATE(:end_date,'YYYY-MM-DD'),TO_DATE(:signed,'YYYY-MM-DD'),:docx,:pdf,:actor) RETURNING id INTO :out_id`, sql.Named("contract", p.ContractID), sql.Named("period", p.PeriodID), sql.Named("template", p.TemplateID), sql.Named("quarter", p.Quarter), sql.Named("start_date", p.Start), sql.Named("end_date", p.End), sql.Named("signed", p.Signed), sql.Named("docx", p.DOCX), sql.Named("pdf", p.PDF), sql.Named("actor", user), sql.Named("out_id", sql.Out{Dest: &p.ID}))
	if err != nil {
		return err
	}
	return tx.Commit()
}
