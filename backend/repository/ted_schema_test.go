package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
)

// This driver exercises database/sql binding and scanning without an Oracle server.
type schemaConnector struct{ conn *schemaConn }

func (c schemaConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (c schemaConnector) Driver() driver.Driver                        { return schemaDriver{} }

type schemaDriver struct{}

func (schemaDriver) Open(string) (driver.Conn, error) { return nil, fmt.Errorf("use connector") }

type schemaConn struct {
	query func(string, []driver.NamedValue) (driver.Rows, error)
	exec  func(string, []driver.NamedValue) (driver.Result, error)
}

func (*schemaConn) Prepare(string) (driver.Stmt, error)      { return nil, fmt.Errorf("unexpected prepare") }
func (*schemaConn) Close() error                             { return nil }
func (*schemaConn) Begin() (driver.Tx, error)                { return nil, fmt.Errorf("unexpected transaction") }
func (*schemaConn) CheckNamedValue(*driver.NamedValue) error { return nil }
func (c *schemaConn) QueryContext(_ context.Context, q string, a []driver.NamedValue) (driver.Rows, error) {
	return c.query(q, a)
}
func (c *schemaConn) ExecContext(_ context.Context, q string, a []driver.NamedValue) (driver.Result, error) {
	return c.exec(q, a)
}

type schemaRows struct {
	values [][]driver.Value
	width  int
}

func (r *schemaRows) Columns() []string {
	cols := make([]string, r.width)
	for i := range cols {
		cols[i] = fmt.Sprint(i)
	}
	return cols
}
func (*schemaRows) Close() error { return nil }
func (r *schemaRows) Next(dest []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(dest, r.values[0])
	r.values = r.values[1:]
	return nil
}
func testSchemaStore(t *testing.T, c *schemaConn) *OracleStore {
	t.Helper()
	db := sql.OpenDB(schemaConnector{conn: c})
	t.Cleanup(func() { db.Close() })
	return &OracleStore{DB: db}
}
func forbidColumns(t *testing.T, q string, cols ...string) {
	t.Helper()
	for _, col := range cols {
		if regexp.MustCompile(`(?i)\b` + col + `\b`).MatchString(q) {
			t.Fatalf("nonexistent column %s in %s", col, q)
		}
	}
}
func checkBinds(t *testing.T, q string, args []driver.NamedValue) {
	t.Helper()
	names := map[string]bool{}
	for _, a := range args {
		names[a.Name] = true
	}
	for _, m := range regexp.MustCompile(`:(\w+)`).FindAllStringSubmatch(q, -1) {
		if !names[m[1]] {
			t.Fatalf("missing bind %s", m[1])
		}
		delete(names, m[1])
	}
	if len(names) > 0 {
		t.Fatalf("unused binds %v", names)
	}
}

func TestTEDUserIdentity(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		forbidColumns(t, q, "sifra", "lozinka")
		if !strings.Contains(q, "FROM TED.UGO_KOR") || args[0].Value != "ad.account" {
			t.Fatal("wrong user lookup")
		}
		return &schemaRows{width: 6, values: [][]driver.Value{{int64(7), "ad.account", "Ime Prezime", "A", nil, nil}}}, nil
	}})
	user, err := store.GetUserByUsername(context.Background(), "ad.account")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "ad.account" || user.ADUsername != user.Username || !user.DateOfCreation.IsZero() {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestTEDInsertUserUsesTriggerID(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{exec: func(q string, args []driver.NamedValue) (driver.Result, error) {
		forbidColumns(t, q, "sifra", "lozinka")
		checkBinds(t, q, args)
		if !strings.Contains(q, "INSERT INTO TED.UGO_KOR") {
			t.Fatal(q)
		}
		for _, a := range args {
			if a.Name == "out0" {
				*a.Value.(sql.Out).Dest.(*int) = 42
			}
		}
		return driver.RowsAffected(1), nil
	}})
	user, err := store.InsertUser(context.Background(), &models.User{ADUsername: "ad.account", Status: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 42 || user.Username != "ad.account" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestTEDContactReadsWithoutOrganization(t *testing.T) {
	for _, paged := range []bool{false, true} {
		t.Run(fmt.Sprint(paged), func(t *testing.T) {
			store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
				forbidColumns(t, q, "id_ugo_org", "ugo_org")
				if !strings.Contains(q, "FROM TED.UGO_DOB_LICA") {
					t.Fatal(q)
				}
				values := []driver.Value{int64(4), "Kontakt", nil, nil, nil, "A", nil, nil, int64(9), "D9", "Dobavljac", nil, nil, nil, nil}
				if paged {
					values = append(values, int64(1))
				}
				return &schemaRows{width: len(values), values: [][]driver.Value{values}}, nil
			}})
			var contact *models.UgoDobLice
			if paged {
				list, total, err := store.GetUgoDobLicePaged(context.Background(), 0, 10, "kontakt")
				if err != nil {
					t.Fatal(err)
				}
				if len(list) != 1 || total != 1 {
					t.Fatal("wrong page")
				}
				contact = list[0]
			} else {
				var err error
				contact, err = store.GetUgoDobLiceById(context.Background(), 4)
				if err != nil {
					t.Fatal(err)
				}
			}
			if contact.ID != 4 || contact.SapDobavljac.ID != 9 || contact.UgoDobLicaRola.ID != 0 || !contact.DatPri.IsZero() {
				t.Fatalf("unexpected contact: %+v", contact)
			}
		})
	}
}

func TestTEDContactWritesWithoutOrganization(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprint(update), func(t *testing.T) {
			store := testSchemaStore(t, &schemaConn{exec: func(q string, args []driver.NamedValue) (driver.Result, error) {
				forbidColumns(t, q, "id_ugo_org")
				checkBinds(t, q, args)
				for _, a := range args {
					if a.Name == "p6" && a.Value != nil {
						t.Fatal("optional role must be NULL")
					}
					if a.Name == "out0" {
						*a.Value.(sql.Out).Dest.(*int) = 8
					}
				}
				return driver.RowsAffected(1), nil
			}})
			contact := &models.UgoDobLice{ID: 8, SapDobavljac: models.SapDobavljac{ID: 9}, Ime: "Kontakt"}
			var err error
			if update {
				_, err = store.UpdateUgoDobLice(context.Background(), contact)
			} else {
				_, err = store.InsertUgoDobLice(context.Background(), contact)
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTEDUpdateAcceptsNullCreationDate(t *testing.T) {
	store := testSchemaStore(t, &schemaConn{exec: func(q string, args []driver.NamedValue) (driver.Result, error) {
		for _, a := range args {
			if out, ok := a.Value.(sql.Out); ok {
				if date, ok := out.Dest.(*sql.NullTime); ok {
					*date = sql.NullTime{}
				}
			}
		}
		return driver.RowsAffected(1), nil
	}})
	org, err := store.UpdateUgoOrg(context.Background(), &models.UgoOrg{Sifra: "ORG", DatPri: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if !org.DatPri.IsZero() {
		t.Fatal("NULL creation date was not preserved")
	}
}

// Guard the explicit SAP read-only requirement across all repository SQL.
func TestSAPTablesAreReadOnlyAndTEDQualified(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	mutations := regexp.MustCompile(`(?i)\b(?:INSERT\s+INTO|UPDATE|DELETE\s+FROM|MERGE\s+INTO)\s+(?:TED\.)?SAP_`)
	unqualified := regexp.MustCompile(`(?i)\b(?:FROM|JOIN|INTO|UPDATE)\s+(?:SAP|UGO)_`)
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if mutations.Match(source) {
			t.Errorf("SAP write in %s", name)
		}
		if unqualified.Match(source) {
			t.Errorf("unqualified table in %s", name)
		}
	}
}
