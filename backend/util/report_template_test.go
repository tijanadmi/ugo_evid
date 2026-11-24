package util

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestReportTemplateProvidedFixture(t *testing.T) {
	path := os.Getenv("REPORT_TEMPLATE_FIXTURE")
	if path == "" {
		t.Skip("optional local DOCX fixture")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = RenderReportTemplate(data, map[string]string{"datum_od": "01.08.2026", "datum_do": "30.11.2026", "sysdate": "01.12.2026", "datum_potpisa": "01.12.2026", "broj_kvartala": "3"}); err != nil {
		t.Fatal(err)
	}
}

func templateZIP(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for n, s := range files {
		w, e := z.Create(n)
		if e != nil {
			t.Fatal(e)
		}
		w.Write([]byte(s))
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestReportTemplateSplitRuns(t *testing.T) {
	data := templateZIP(t, map[string]string{"[Content_Types].xml": `<Types/>`, "word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>Период {{datum_</w:t></w:r><w:r><w:t>od}} — {{datum_do}}; {{broj_kvartala}}. {{datum_od}}</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>RUN-123</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`, "word/header1.xml": `<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>{{datum_potpisa}}</w:t></w:r></w:p></w:hdr>`})
	out, err := RenderReportTemplate(data, map[string]string{"datum_od": "01.08.2026", "datum_do": "30.11.2026", "datum_potpisa": "01.12.2026", "broj_kvartala": "5 & 6"})
	if err != nil {
		t.Fatal(err)
	}
	z, _ := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	var all string
	for _, f := range z.File {
		r, _ := f.Open()
		b, _ := io.ReadAll(r)
		r.Close()
		all += string(b)
	}
	for _, want := range []string{"01.08.2026", "30.11.2026", "01.12.2026", "5 &amp; 6", "<w:b/>", "RUN-123", "<w:tbl>"} {
		if !strings.Contains(all, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if strings.Contains(all, "{{") {
		t.Fatal("unresolved marker")
	}
}
func TestReportTemplateRejectsUnsafeOrUnknown(t *testing.T) {
	valid := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>{{datum_od}} {{datum_do}} {{sysdate}}</w:t></w:r></w:p></w:document>`
	values := map[string]string{"datum_od": "a", "datum_do": "b", "sysdate": "c"}
	for _, quarter := range []bool{false, true} {
		doc := valid
		if quarter {
			doc = strings.Replace(doc, "{{datum_od}}", "{{broj_</w:t></w:r><w:r><w:t>kvartala}} {{datum_od}}", 1)
		}
		got, err := TemplateUsesQuarter(templateZIP(t, map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": doc}))
		if err != nil || got != quarter {
			t.Fatalf("quarter=%v got=%v err=%v", quarter, got, err)
		}
	}
	for _, mode := range []string{"unknown", "missing", "external", "field", "macro", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			files := map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": valid}
			switch mode {
			case "unknown":
				files["word/document.xml"] = strings.Replace(valid, "datum_od", "secret", 1)
			case "missing":
				files["word/document.xml"] = "<doc/>"
			case "external":
				files["word/_rels/document.xml.rels"] = `<Relationships><Relationship TargetMode="External" Target="file:///secret"/></Relationships>`
			case "field":
				files["word/document.xml"] = strings.Replace(valid, "<w:p>", "<w:p><w:instrText>DDE</w:instrText>", 1)
			case "macro":
				files["word/vbaProject.bin"] = "macro"
			case "invalid":
				files["word/document.xml"] = "<broken>"
			}
			if _, e := RenderReportTemplate(templateZIP(t, files), values); e == nil {
				t.Fatal("accepted unsafe template")
			}
		})
	}
	if _, err := ReportPDF(context.Background(), "", nil); err == nil {
		t.Fatal("missing converter accepted")
	}
}
