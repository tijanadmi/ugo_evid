-- Optional calendar preparation, after maintenance_tracking.sql.
-- SQL*Plus / SQLcl: set the year below. Repeat for each needed year.
-- Creates PROVISIONAL weekdays/weekends. DOES NOT assume public holidays.
-- Existing dates and approvals are preserved.
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK
DEFINE GODINA = 2026
MERGE INTO TED.UGO_RADNI_KALENDAR k
USING (
 SELECT TO_DATE('&&GODINA-01-01','YYYY-MM-DD')+LEVEL-1 datum
 FROM dual CONNECT BY LEVEL <=
 TO_DATE('&&GODINA-12-31','YYYY-MM-DD')-TO_DATE('&&GODINA-01-01','YYYY-MM-DD')+1
) s ON (k.datum=s.datum)
WHEN NOT MATCHED THEN INSERT (datum,radni,potvrdjen,opis)
VALUES(s.datum,CASE WHEN s.datum-TRUNC(s.datum,'IW') IN (5,6) THEN 'N' ELSE 'D' END,'N','Predlog - proveriti praznike');
COMMIT;
-- Correct all public/company holidays and exceptional working weekends first.
-- Example only (replace the date and description with approved calendar data):
-- UPDATE TED.UGO_RADNI_KALENDAR SET radni='N',opis='Neradni dan'
-- WHERE datum=DATE '2026-01-01';
-- After reviewing the ENTIRE year, explicitly approve it:
-- UPDATE TED.UGO_RADNI_KALENDAR SET potvrdjen='D'
-- WHERE datum>=DATE '2026-01-01' AND datum<DATE '2027-01-01';
-- COMMIT;
SELECT potvrdjen,COUNT(*) broj FROM TED.UGO_RADNI_KALENDAR
WHERE datum>=TO_DATE('&&GODINA-01-01','YYYY-MM-DD')
AND datum<=TO_DATE('&&GODINA-12-31','YYYY-MM-DD') GROUP BY potvrdjen;
