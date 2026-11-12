-- Run as TED after portal_admin.sql, only for the FIRST administrator.
-- Replace the placeholder with an EXISTING active UGO_KOR.AD_SIFRA.
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK
DECLARE
  v_username VARCHAR2(200) := 'replace.with.actual.ad.username';
  v_active VARCHAR2(1) := 'A'; -- must match ACTIVE_USER_STATUS
  v_id NUMBER;
  v_count NUMBER;
  v_lock NUMBER;
BEGIN
  IF v_username='replace.with.actual.ad.username' THEN
    raise_application_error(-20001,'Set the existing AD username first.');
  END IF;
  SELECT id INTO v_lock FROM TED.PORTAL_ADMIN_LOCK WHERE id=1 FOR UPDATE WAIT 5;
  SELECT COUNT(*) INTO v_count FROM TED.UGO_KOR k
  WHERE k.status=v_active AND EXISTS (
    SELECT 1 FROM TED.UGO_KOR_ROLE kr WHERE kr.id_ugo_kor=k.id
    AND kr.id_ugo_rola=4 AND kr.opseg='GLOBALNO' AND kr.status='A');
  IF v_count>0 THEN
    raise_application_error(-20002,'An active ADMIN already exists. Use portal administration.');
  END IF;
  SELECT id INTO v_id FROM TED.UGO_KOR
  WHERE LOWER(TRIM(ad_sifra))=LOWER(TRIM(v_username)) AND status=v_active;
  INSERT INTO TED.UGO_KOR_ROLE (id_ugo_kor,id_ugo_rola,opseg,status,datpri)
  VALUES (v_id,4,'GLOBALNO','A',SYSDATE);
  UPDATE TED.UGO_KOR SET version=version+1,datizm=SYSDATE WHERE id=v_id;
  INSERT INTO TED.PORTAL_ADMIN_AUDIT (id_korisnika,akcija,vrsta,id_sloga,pre,posle)
  VALUES (v_id,'BOOTSTRAP_ADMIN','KORISNIK',v_id,NULL,'{"added_role":"ADMIN","opseg":"GLOBALNO"}');
  COMMIT;
EXCEPTION
  WHEN OTHERS THEN
    ROLLBACK;
    RAISE;
END;
/
