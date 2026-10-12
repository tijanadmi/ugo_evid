// doc-reconcile reports pending uploads; -apply activates only complete,
// checksum-verified files older than an hour. Run with uploads stopped.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/sijms/go-ora/v2"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"os"
	"time"
)

func run() error {
	apply := flag.Bool("apply", false, "activate verified pending documents; default is report only")
	flag.Parse()
	config, err := util.LoadConfig(".")
	if err != nil {
		return err
	}
	db, err := sql.Open("oracle", config.DBSource)
	if err != nil {
		return err
	}
	defer db.Close()
	root, err := util.DocumentStorage(config.DocumentRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	store := &repository.OracleStore{DB: db}
	docs, err := store.PendingDocuments(ctx)
	if err != nil {
		return err
	}
	for _, doc := range docs {
		if err := util.VerifyDocument(root, doc.Path, doc.SHA256, doc.Size); err != nil {
			fmt.Printf("ID %d: incomplete/missing file; retained for review (%v)\n", doc.ID, err)
			continue
		}
		if *apply {
			if err := store.FinalizeDocument(ctx, doc.ID); err != nil {
				return err
			}
			fmt.Printf("ID %d: activated\n", doc.ID)
		} else {
			fmt.Printf("ID %d: complete, eligible for activation\n", doc.ID)
		}
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
