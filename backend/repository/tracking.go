package repository

import (
	"context"
	"database/sql"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"time"
	"unicode/utf8"
)

func (r *OracleStore) GetTracking(ctx context.Context, id int) (*models.TrackingData, error) {
	out := &models.TrackingData{Periods: []models.TrackingItem{}, Requests: []models.TrackingItem{}, Funds: []models.FundBalance{}}
	rows, err := r.DB.QueryContext(ctx, `SELECT p.id,p.version,TO_CHAR(p.period_od,'YYYY-MM-DD'),TO_CHAR(p.period_do,'YYYY-MM-DD'),p.id_tip_doc,TO_CHAR(p.rok_dokumenta,'YYYY-MM-DD'),TO_CHAR(p.datum_potpisa,'YYYY-MM-DD'),NVL(p.id_ugo_doc,0),p.status,p.napomena FROM TED.UGO_TEH_PERIOD p JOIN TED.UGO_ODRZAVANJE o ON o.id=p.id_odrzavanja WHERE o.id_ugo_evid=:id ORDER BY p.period_od,p.id`, sql.Named("id", id))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p models.TrackingItem
		if err = scanNullable(rows, &p.ID, &p.Version, &p.Start, &p.End, &p.DocumentType, &p.Deadline, &p.Signed, &p.Document, &p.Status, &p.Note); err != nil {
			rows.Close()
			return nil, err
		}
		out.Periods = append(out.Periods, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = r.DB.QueryContext(ctx, `SELECT z.id,z.version,f.id_vrsta_rada,z.broj_zahteva,TO_CHAR(z.datum_zahteva,'YYYY-MM-DD'),z.broj_casova,z.rok_radnih_dana,TO_CHAR(z.rok_potpisa,'YYYY-MM-DD'),TO_CHAR(z.datum_potpisa,'YYYY-MM-DD'),NVL(z.id_ugo_doc,0),NVL(z.id_zahtev_doc,0),z.status,z.napomena FROM TED.UGO_RAZ_ZAHTEV z JOIN TED.UGO_RAZ_FOND f ON f.id=z.id_raz_fonda JOIN TED.UGO_ODRZAVANJE o ON o.id=f.id_odrzavanja WHERE o.id_ugo_evid=:id ORDER BY z.datum_zahteva,z.id`, sql.Named("id", id))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p models.TrackingItem
		if err = scanNullable(rows, &p.ID, &p.Version, &p.WorkType, &p.Number, &p.Requested, &p.Hours, &p.Days, &p.Deadline, &p.Signed, &p.Document, &p.RequestDocument, &p.Status, &p.Note); err != nil {
			rows.Close()
			return nil, err
		}
		out.Requests = append(out.Requests, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = r.DB.QueryContext(ctx, `SELECT f.id_vrsta_rada,v.naziv,f.ugovoreno_casova,NVL(SUM(CASE WHEN z.status NOT IN ('NACRT','OTKAZAN') THEN z.broj_casova ELSE 0 END),0) FROM TED.UGO_RAZ_FOND f JOIN TED.UGO_RAZ_VRSTA_RADA v ON v.id=f.id_vrsta_rada JOIN TED.UGO_ODRZAVANJE o ON o.id=f.id_odrzavanja LEFT JOIN TED.UGO_RAZ_ZAHTEV z ON z.id_raz_fonda=f.id WHERE o.id_ugo_evid=:id AND f.ugovoreno_casova>0 GROUP BY f.id_vrsta_rada,v.naziv,f.ugovoreno_casova ORDER BY f.id_vrsta_rada`, sql.Named("id", id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f models.FundBalance
		if err = rows.Scan(&f.WorkType, &f.Name, &f.Contracted, &f.Used); err != nil {
			return nil, err
		}
		f.Remaining = f.Contracted - f.Used
		out.Funds = append(out.Funds, f)
	}
	return out, rows.Err()
}

func (r *OracleStore) PreviewPeriods(ctx context.Context, id int, mode string) ([]models.TrackingItem, error) {
	s, err := r.GetContractSettings(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.Maintenance == nil || s.Maintenance.Months == nil {
		return nil, ErrTrackingInput
	}
	start := s.Maintenance.Start
	if start == "" {
		start = s.SAPStart
	}
	return GeneratePeriods(start, s.SAPEnd, *s.Maintenance.Months, s.Maintenance.DocumentType, mode)
}

func (r *OracleStore) SaveTracking(ctx context.Context, id, org, user int, kind string, change models.TrackingChange) error {
	if (kind != "periodi" && kind != "zahtevi") || len(change.Items) == 0 || len(change.Items) > 1200 || (kind == "zahtevi" && len(change.Items) != 1) {
		return ErrTrackingInput
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, _, err = lockMaintenanceParent(ctx, tx, id, org); err != nil {
		return err
	}
	if err = requireMaintenanceLease(ctx, tx, id, user, change.Token); err != nil {
		return err
	}
	var mid, development int
	var start, end string
	err = scanNullable(tx.QueryRowContext(ctx, `SELECT o.id,o.ima_razvojnu_podrsku,TO_CHAR(NVL(o.pocetak_odrzavanja,s.pocetak_ug),'YYYY-MM-DD'),TO_CHAR(s.kraj_ug,'YYYY-MM-DD') FROM TED.UGO_ODRZAVANJE o JOIN TED.UGO_EVID e ON e.id=o.id_ugo_evid JOIN TED.SAP_UGOVORI s ON s.id=e.id_sap_ugovor WHERE e.id=:id`, sql.Named("id", id)), &mid, &development, &start, &end)
	if err != nil {
		return err
	}
	if start == "" || end == "" {
		return ErrTrackingInput
	}
	seen := map[int]bool{}
	for _, p := range change.Items {
		if p.ID < 0 || (p.ID > 0 && (seen[p.ID] || p.Version < 1)) || utf8.RuneCountInString(p.Note) > 2000 {
			return ErrTrackingInput
		}
		seen[p.ID] = true
		if p.Signed != "" {
			d, e := date(p.Signed)
			if e != nil || d.After(time.Now().UTC().Truncate(24*time.Hour)) {
				return ErrTrackingInput
			}
		}
		if p.Status == "ZAVRSEN" && p.Signed == "" {
			return ErrTrackingInput
		}
		if p.Document < 0 || p.RequestDocument < 0 {
			return ErrTrackingInput
		}
		for _, doc := range []int{p.Document, p.RequestDocument} {
			if doc == 0 {
				continue
			}
			var count int
			err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_DOC WHERE id=:doc AND id_ugo_evid=:id AND status='A'`, sql.Named("doc", doc), sql.Named("id", id)).Scan(&count)
			if err != nil {
				return err
			}
			if count != 1 {
				return ErrTrackingInput
			}
		}
		if kind == "periodi" {
			err = savePeriod(ctx, tx, mid, user, start, end, p)
		} else {
			if development != 1 {
				return ErrTrackingInput
			}
			err = saveRequest(ctx, tx, mid, user, start, end, p)
		}
		if err != nil {
			if strings.Contains(err.Error(), "ORA-00001") {
				return ErrMaintenanceConflict
			}
			return err
		}
	}
	if err = releaseMaintenanceLease(ctx, tx, id, user, change.Token); err != nil {
		return err
	}
	return tx.Commit()
}

func savePeriod(ctx context.Context, tx *sql.Tx, mid, user int, start, end string, p models.TrackingItem) error {
	if _, e := date(p.Start); e != nil {
		return ErrTrackingInput
	}
	if _, e := date(p.End); e != nil {
		return ErrTrackingInput
	}
	if p.Start > p.End || p.Start < start || p.End > end || !strings.Contains("|PLANIRAN|PRIPREMA|NA_POTPISU|ZAVRSEN|OTKAZAN|", "|"+p.Status+"|") {
		return ErrTrackingInput
	}
	if p.Deadline != "" {
		if _, e := date(p.Deadline); e != nil || p.Deadline < p.End {
			return ErrTrackingInput
		}
	}
	if p.Signed != "" && p.Signed < p.End {
		return ErrTrackingInput
	}
	var count int
	if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_TIP_DOC WHERE id=:id AND status='A' AND sifra IN ('POTVRDA_TEH','ZAPISNIK_TEH')`, sql.Named("id", p.DocumentType)).Scan(&count); e != nil {
		return e
	}
	if count != 1 {
		return ErrTrackingInput
	}
	if p.Status != "OTKAZAN" {
		e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_TEH_PERIOD WHERE id_odrzavanja=:mid AND id<>:id AND status<>'OTKAZAN' AND period_od<=TO_DATE(:end_date,'YYYY-MM-DD') AND period_do>=TO_DATE(:start_date,'YYYY-MM-DD')`, sql.Named("mid", mid), sql.Named("id", p.ID), sql.Named("end_date", p.End), sql.Named("start_date", p.Start)).Scan(&count)
		if e != nil {
			return e
		}
		if count > 0 {
			return ErrTrackingInput
		}
	}
	args := []any{sql.Named("start_date", p.Start), sql.Named("end_date", p.End), sql.Named("dtype", p.DocumentType), sql.Named("deadline", p.Deadline), sql.Named("signed", p.Signed), sql.Named("doc", p.Document), sql.Named("status", p.Status), sql.Named("note", p.Note), sql.Named("user_id", user), sql.Named("mid", mid)}
	if p.ID == 0 {
		_, e := tx.ExecContext(ctx, `INSERT INTO TED.UGO_TEH_PERIOD(period_od,period_do,id_tip_doc,rok_dokumenta,datum_potpisa,id_ugo_doc,status,napomena,id_korisnika,id_odrzavanja) VALUES(TO_DATE(:start_date,'YYYY-MM-DD'),TO_DATE(:end_date,'YYYY-MM-DD'),:dtype,TO_DATE(:deadline,'YYYY-MM-DD'),TO_DATE(:signed,'YYYY-MM-DD'),NULLIF(:doc,0),:status,:note,:user_id,:mid)`, args...)
		return e
	}
	args = append(args, sql.Named("id", p.ID), sql.Named("version", p.Version))
	result, e := tx.ExecContext(ctx, `UPDATE TED.UGO_TEH_PERIOD SET period_od=TO_DATE(:start_date,'YYYY-MM-DD'),period_do=TO_DATE(:end_date,'YYYY-MM-DD'),id_tip_doc=:dtype,rok_dokumenta=TO_DATE(:deadline,'YYYY-MM-DD'),datum_potpisa=TO_DATE(:signed,'YYYY-MM-DD'),id_ugo_doc=NULLIF(:doc,0),status=:status,napomena=:note,id_korisnika=:user_id,datizm=SYSTIMESTAMP,version=version+1 WHERE id_odrzavanja=:mid AND id=:id AND version=:version`, args...)
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrMaintenanceConflict
	}
	return nil
}

func saveRequest(ctx context.Context, tx *sql.Tx, mid, user int, start, end string, p models.TrackingItem) error {
	if _, e := date(p.Requested); e != nil {
		return ErrTrackingInput
	}
	if p.Requested < start || p.Requested > end || p.Hours < 1 || p.Hours > 2147483647 || p.Days < 1 || p.Days > 3660 || strings.TrimSpace(p.Number) == "" || utf8.RuneCountInString(p.Number) > 100 || !strings.Contains("|NACRT|PODNET|U_RADU|NA_POTPISU|ZAVRSEN|OTKAZAN|", "|"+p.Status+"|") {
		return ErrTrackingInput
	}
	var fid, contracted, used int
	e := tx.QueryRowContext(ctx, `SELECT id,ugovoreno_casova FROM TED.UGO_RAZ_FOND WHERE id_odrzavanja=:mid AND id_vrsta_rada=:work FOR UPDATE NOWAIT`, sql.Named("mid", mid), sql.Named("work", p.WorkType)).Scan(&fid, &contracted)
	if e == sql.ErrNoRows {
		return ErrTrackingFunds
	}
	if e != nil {
		return e
	}
	if contracted <= 0 {
		return ErrTrackingFunds
	}
	e = tx.QueryRowContext(ctx, `SELECT NVL(SUM(broj_casova),0) FROM TED.UGO_RAZ_ZAHTEV WHERE id_raz_fonda=:fid AND id<>:id AND status NOT IN ('NACRT','OTKAZAN')`, sql.Named("fid", fid), sql.Named("id", p.ID)).Scan(&used)
	if e != nil {
		return e
	}
	if p.Status != "NACRT" && p.Status != "OTKAZAN" && int64(used)+int64(p.Hours) > int64(contracted) {
		return ErrTrackingFunds
	}
	// Keep the agreed deadline snapshot when only status/documents change.
	oldDate, deadline := "", ""
	oldDays := 0
	if p.ID > 0 {
		e = scanNullable(tx.QueryRowContext(ctx, `SELECT z.rok_radnih_dana,TO_CHAR(z.datum_zahteva,'YYYY-MM-DD'),TO_CHAR(z.rok_potpisa,'YYYY-MM-DD') FROM TED.UGO_RAZ_ZAHTEV z JOIN TED.UGO_RAZ_FOND f ON f.id=z.id_raz_fonda WHERE z.id=:id AND z.version=:version AND f.id_odrzavanja=:mid`, sql.Named("id", p.ID), sql.Named("version", p.Version), sql.Named("mid", mid)), &oldDays, &oldDate, &deadline)
		if e == sql.ErrNoRows {
			return ErrMaintenanceConflict
		}
		if e != nil {
			return e
		}
	}
	if p.Requested != oldDate || p.Days != oldDays {
		rows, e := tx.QueryContext(ctx, `SELECT TO_CHAR(datum,'YYYY-MM-DD'),radni FROM TED.UGO_RADNI_KALENDAR WHERE potvrdjen='D' AND datum>TO_DATE(:start_date,'YYYY-MM-DD') AND datum<=TO_DATE(:limit_date,'YYYY-MM-DD')+20000 ORDER BY datum`, sql.Named("start_date", p.Requested), sql.Named("limit_date", p.Requested))
		if e != nil {
			return e
		}
		calendar := map[string]bool{}
		for rows.Next() {
			var d, w string
			if e = rows.Scan(&d, &w); e != nil {
				rows.Close()
				return e
			}
			calendar[d] = w == "D"
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		deadline, e = requestDeadline(p.Requested, p.Days, calendar)
		if e != nil {
			return e
		}
	}
	if p.Signed != "" && (p.Signed < p.Requested || p.Signed >= deadline) {
		return ErrTrackingInput
	}
	args := []any{sql.Named("fid", fid), sql.Named("number", strings.TrimSpace(p.Number)), sql.Named("requested", p.Requested), sql.Named("hours", p.Hours), sql.Named("days", p.Days), sql.Named("deadline", deadline), sql.Named("signed", p.Signed), sql.Named("doc", p.Document), sql.Named("request_doc", p.RequestDocument), sql.Named("status", p.Status), sql.Named("note", p.Note), sql.Named("user_id", user)}
	if p.ID == 0 {
		_, e = tx.ExecContext(ctx, `INSERT INTO TED.UGO_RAZ_ZAHTEV(id_raz_fonda,broj_zahteva,datum_zahteva,broj_casova,rok_radnih_dana,rok_potpisa,datum_potpisa,id_ugo_doc,id_zahtev_doc,status,napomena,id_korisnika) VALUES(:fid,:number,TO_DATE(:requested,'YYYY-MM-DD'),:hours,:days,TO_DATE(:deadline,'YYYY-MM-DD'),TO_DATE(:signed,'YYYY-MM-DD'),NULLIF(:doc,0),NULLIF(:request_doc,0),:status,:note,:user_id)`, args...)
		return e
	}
	args = append(args, sql.Named("id", p.ID), sql.Named("version", p.Version))
	result, e := tx.ExecContext(ctx, `UPDATE TED.UGO_RAZ_ZAHTEV SET id_raz_fonda=:fid,broj_zahteva=:number,datum_zahteva=TO_DATE(:requested,'YYYY-MM-DD'),broj_casova=:hours,rok_radnih_dana=:days,rok_potpisa=TO_DATE(:deadline,'YYYY-MM-DD'),datum_potpisa=TO_DATE(:signed,'YYYY-MM-DD'),id_ugo_doc=NULLIF(:doc,0),id_zahtev_doc=NULLIF(:request_doc,0),status=:status,napomena=:note,id_korisnika=:user_id,datizm=SYSTIMESTAMP,version=version+1 WHERE id=:id AND version=:version`, args...)
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrMaintenanceConflict
	}
	return nil
}

func (r *OracleStore) ListReminders(ctx context.Context, org int) ([]models.Reminder, error) {
	rows, e := r.DB.QueryContext(ctx, `SELECT id_ugo_evid,vrsta,id,opis,TO_CHAR(rok,'YYYY-MM-DD'),kasni FROM (
 SELECT e.id id_ugo_evid,'TEH' vrsta,p.id,TO_CHAR(p.period_od,'DD.MM.YYYY')||' - '||TO_CHAR(p.period_do,'DD.MM.YYYY') opis,NVL(p.rok_dokumenta,p.period_do) rok,CASE WHEN p.rok_dokumenta IS NOT NULL AND TRUNC(SYSDATE)>p.rok_dokumenta THEN 1 ELSE 0 END kasni
 FROM TED.UGO_TEH_PERIOD p JOIN TED.UGO_ODRZAVANJE o ON o.id=p.id_odrzavanja JOIN TED.UGO_EVID e ON e.id=o.id_ugo_evid JOIN TED.SAP_UGOVORI s ON s.id=e.id_sap_ugovor
 WHERE e.id_ugo_org=:org AND s.otvoren_ug='X' AND p.status NOT IN ('ZAVRSEN','OTKAZAN') AND TRUNC(SYSDATE)>=p.period_do-o.teh_podsetnik_dana
 UNION ALL
 SELECT e.id,'RAZ',z.id,z.broj_zahteva,z.rok_potpisa,CASE WHEN TRUNC(SYSDATE)>=z.rok_potpisa THEN 1 ELSE 0 END
 FROM TED.UGO_RAZ_ZAHTEV z JOIN TED.UGO_RAZ_FOND f ON f.id=z.id_raz_fonda JOIN TED.UGO_ODRZAVANJE o ON o.id=f.id_odrzavanja JOIN TED.UGO_EVID e ON e.id=o.id_ugo_evid JOIN TED.SAP_UGOVORI s ON s.id=e.id_sap_ugovor
 WHERE e.id_ugo_org=:org2 AND s.otvoren_ug='X' AND z.status NOT IN ('NACRT','ZAVRSEN','OTKAZAN') AND TRUNC(SYSDATE)>=z.rok_potpisa-o.raz_podsetnik_dana
 ) ORDER BY rok,id_ugo_evid,id`, sql.Named("org", org), sql.Named("org2", org))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []models.Reminder{}
	for rows.Next() {
		var p models.Reminder
		var late int
		if e = rows.Scan(&p.ContractID, &p.Kind, &p.ID, &p.Label, &p.Due, &late); e != nil {
			return nil, e
		}
		p.Overdue = late == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

// Called in the same parent-locked transaction as maintenance settings edits.
func validateTrackingSettings(ctx context.Context, tx *sql.Tx, mid int, start, end string, m models.Maintenance) error {
	var count int
	e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_TEH_PERIOD WHERE id_odrzavanja=:mid AND status<>'OTKAZAN' AND (period_od<TO_DATE(:start_date,'YYYY-MM-DD') OR period_do>TO_DATE(:end_date,'YYYY-MM-DD'))`, sql.Named("mid", mid), sql.Named("start_date", start), sql.Named("end_date", end)).Scan(&count)
	if e != nil {
		return e
	}
	if count > 0 {
		return ErrTrackingDependencies
	}
	rows, e := tx.QueryContext(ctx, `SELECT f.id_vrsta_rada,NVL(SUM(CASE WHEN z.status NOT IN ('NACRT','OTKAZAN') THEN z.broj_casova ELSE 0 END),0),SUM(CASE WHEN z.status<>'OTKAZAN' THEN 1 ELSE 0 END),SUM(CASE WHEN z.status<>'OTKAZAN' AND (z.datum_zahteva<TO_DATE(:start_date,'YYYY-MM-DD') OR z.datum_zahteva>TO_DATE(:end_date,'YYYY-MM-DD')) THEN 1 ELSE 0 END) FROM TED.UGO_RAZ_FOND f JOIN TED.UGO_RAZ_ZAHTEV z ON z.id_raz_fonda=f.id WHERE f.id_odrzavanja=:mid GROUP BY f.id_vrsta_rada`, sql.Named("start_date", start), sql.Named("end_date", end), sql.Named("mid", mid))
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var work, used, active, outside int
		if e = rows.Scan(&work, &used, &active, &outside); e != nil {
			return e
		}
		hours := 0
		for _, f := range m.Funds {
			if f.TypeID == work {
				hours = f.Hours
			}
		}
		if used > hours || outside > 0 || (active > 0 && (!m.Development || hours == 0)) {
			return ErrTrackingDependencies
		}
	}
	return rows.Err()
}
