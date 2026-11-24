package util

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const TemplateMaxBytes = 10 << 20

var reportPDFSlots = make(chan struct{}, 2)

var ErrTemplate = errors.New("Neispravan DOCX šablon. Koristite oznake datum_od, datum_do, datum_potpisa i broj_kvartala; bez makroa, spoljnih veza i Word polja.")
var paragraphRE = regexp.MustCompile(`(?s)<w:p(?:\s[^>]*)?>.*?</w:p>`)
var textRE = regexp.MustCompile(`(?s)<w:t(?:\s[^>]*)?>(.*?)</w:t>`)
var markerRE = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

func xmlText(s string) (string, error) {
	var v struct {
		Text string `xml:",chardata"`
	}
	err := xml.Unmarshal([]byte("<t>"+s+"</t>"), &v)
	return v.Text, err
}
func escapeXML(s string) string { var b bytes.Buffer; xml.EscapeText(&b, []byte(s)); return b.String() }

// Replace visible text only, including markers split by Word across styled runs.
// Preserve all other XML bytes, drawings, tables and run properties.
func replaceParagraph(p string, values map[string]string, seen map[string]bool) (string, error) {
	runs := textRE.FindAllStringSubmatchIndex(p, -1)
	texts := make([]string, len(runs))
	starts := make([]int, len(runs))
	var all strings.Builder
	for i, r := range runs {
		var err error
		texts[i], err = xmlText(p[r[2]:r[3]])
		if err != nil {
			return "", ErrTemplate
		}
		starts[i] = all.Len()
		all.WriteString(texts[i])
	}
	joined := all.String()
	matches := markerRE.FindAllStringSubmatchIndex(joined, -1)
	for m := len(matches) - 1; m >= 0; m-- {
		x := matches[m]
		key := joined[x[2]:x[3]]
		value, ok := values[key]
		if !ok {
			return "", fmt.Errorf("%w Nepoznata oznaka: %s", ErrTemplate, key)
		}
		seen[key] = true
		first, last := -1, -1
		for i := range runs {
			end := starts[i] + len(texts[i])
			if first < 0 && x[0] >= starts[i] && x[0] < end {
				first = i
			}
			if x[1] > starts[i] && x[1] <= end {
				last = i
				break
			}
		}
		if first < 0 || last < 0 {
			return "", ErrTemplate
		}
		if first == last {
			texts[first] = texts[first][:x[0]-starts[first]] + value + texts[first][x[1]-starts[first]:]
		} else {
			suffix := texts[last][x[1]-starts[last]:]
			texts[first] = texts[first][:x[0]-starts[first]] + value
			for i := first + 1; i < last; i++ {
				texts[i] = ""
			}
			texts[last] = suffix
		}
	}
	for i := len(runs) - 1; i >= 0; i-- {
		r := runs[i]
		p = p[:r[2]] + escapeXML(texts[i]) + p[r[3]:]
	}
	return p, nil
}

func RenderReportTemplate(data []byte, values map[string]string) ([]byte, error) {
	return renderReportTemplate(data, values, map[string]bool{})
}

func TemplateUsesQuarter(data []byte) (bool, error) {
	seen := map[string]bool{}
	_, err := renderReportTemplate(data, map[string]string{"datum_od": "01.01.2026", "datum_do": "31.01.2026", "datum_potpisa": "01.02.2026", "sysdate": "01.02.2026", "broj_kvartala": "1"}, seen)
	return seen["broj_kvartala"], err
}

func renderReportTemplate(data []byte, values map[string]string, seen map[string]bool) ([]byte, error) {
	if len(data) == 0 || len(data) > TemplateMaxBytes {
		return nil, ErrTemplate
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(z.File) > 2000 {
		return nil, ErrTemplate
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	total := int64(0)
	names := map[string]bool{}
	hasDoc := false
	for _, f := range z.File {
		name := f.Name
		lower := strings.ToLower(name)
		if names[name] || strings.Contains(name, "..") || strings.HasPrefix(name, "/") || strings.Contains(name, `\`) || strings.Contains(lower, "vba") || strings.Contains(lower, "embeddings/") || strings.HasSuffix(lower, ".bin") {
			return nil, ErrTemplate
		}
		names[name] = true
		r, e := f.Open()
		if e != nil {
			return nil, ErrTemplate
		}
		b, e := io.ReadAll(io.LimitReader(r, 32<<20+1))
		r.Close()
		total += int64(len(b))
		if e != nil || total > 32<<20 {
			return nil, ErrTemplate
		}
		if strings.HasSuffix(lower, ".xml") || strings.HasSuffix(lower, ".rels") {
			dec := xml.NewDecoder(bytes.NewReader(b))
			for {
				tok, e := dec.Token()
				if e == io.EOF {
					break
				}
				if e != nil {
					return nil, ErrTemplate
				}
				switch t := tok.(type) {
				case xml.Directive:
					return nil, ErrTemplate
				case xml.StartElement:
					switch strings.ToLower(t.Name.Local) {
					case "altchunk", "instrtext", "fldsimple", "object":
						return nil, ErrTemplate
					}
					for _, a := range t.Attr {
						if (a.Name.Local == "TargetMode" && strings.EqualFold(a.Value, "External")) || strings.Contains(strings.ToLower(a.Value), "macroenabled") {
							return nil, ErrTemplate
						}
					}
				}
			}
		}
		if name == "word/document.xml" {
			hasDoc = true
		}
		if strings.HasPrefix(name, "word/") && strings.HasSuffix(name, ".xml") {
			var replaceErr error
			b = paragraphRE.ReplaceAllFunc(b, func(p []byte) []byte {
				v, e := replaceParagraph(string(p), values, seen)
				if e != nil {
					replaceErr = e
				}
				return []byte(v)
			})
			if replaceErr != nil {
				return nil, replaceErr
			}
			// Reject markers in unsupported parts rather than silently leaving them.
			if bytes.Contains(b, []byte("{{")) || bytes.Contains(b, []byte("}}")) {
				return nil, ErrTemplate
			}
		}
		w, e := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(b); e != nil {
			return nil, e
		}
	}
	if !hasDoc || !names["[Content_Types].xml"] || !seen["datum_od"] || !seen["datum_do"] || (!seen["datum_potpisa"] && !seen["sysdate"]) {
		return nil, ErrTemplate
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Each conversion has a private profile, bounded runtime and a generated filename.
func ReportPDF(ctx context.Context, executable string, docx []byte) ([]byte, error) {
	if strings.TrimSpace(executable) == "" {
		return nil, errors.New("PDF konverzija nije podešena (LIBREOFFICE_PATH).")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	select {
	case reportPDFSlots <- struct{}{}:
		defer func() { <-reportPDFSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	dir, err := os.MkdirTemp("", "ugo-report-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "report.docx")
	if err = os.WriteFile(input, docx, 0600); err != nil {
		return nil, err
	}
	profile := filepath.ToSlash(filepath.Join(dir, "profile"))
	if !strings.HasPrefix(profile, "/") {
		profile = "/" + profile
	}
	uri := (&url.URL{Scheme: "file", Path: profile}).String()
	cmd := exec.CommandContext(ctx, executable, "-env:UserInstallation="+uri, "--headless", "--nologo", "--norestore", "--convert-to", "pdf:writer_pdf_Export", "--outdir", dir, input)
	cmd.WaitDelay = 2 * time.Second
	if err = cmd.Run(); err != nil {
		return nil, errors.New("PDF konverzija nije uspela. Proverite LibreOffice na serveru.")
	}
	f, err := os.Open(filepath.Join(dir, "report.pdf"))
	if err != nil {
		return nil, errors.New("Konverter nije kreirao PDF.")
	}
	defer f.Close()
	pdf, err := io.ReadAll(io.LimitReader(f, 32<<20+1))
	if err != nil || len(pdf) > 32<<20 || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return nil, errors.New("Konverter nije vratio ispravan PDF.")
	}
	return pdf, nil
}
