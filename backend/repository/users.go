package repository

import (
	"context"
	"fmt"
	"time"

	"database/sql"
	"github.com/tijanadmi/ugo_evid/models"
)

// Get returns a user by username
func (r *OracleStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT
id AS c0,
ad_sifra AS c1,
ime AS c3,
status AS c4,
datpri AS c5,
datizm AS c6
 FROM TED.UGO_KOR
		WHERE LOWER(ad_sifra) = LOWER(:p1)
	`

	var dbUser models.User

	row, err := r.DB.QueryContext(ctx, query, sql.Named("p1", username))
	if err != nil {
		return nil, err
	}
	defer row.Close()
	if !row.Next() {
		return nil, row.Err()
	}
	err = scanNullable(row,
		&dbUser.ID,
		&dbUser.ADUsername,
		&dbUser.FullName,
		&dbUser.Status,
		&dbUser.DateOfCreation,
		&dbUser.DateOfLastUpdate,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// korisnik ne postoji
			return nil, nil
		}
		return nil, err
	}

	if row.Next() {
		return nil, fmt.Errorf("ambiguous application identity")
	}
	if err := row.Err(); err != nil {
		return nil, err
	}
	dbUser.Username = dbUser.ADUsername
	return &dbUser, nil
}

// InsertUser ubacuje novog korisnika u bazu podataka
func (r *OracleStore) InsertUser(ctx context.Context, user *models.User) (*models.User, error) {
	query := `
		INSERT INTO TED.UGO_KOR
			(ad_sifra, ime, status, datpri, datizm)
		VALUES 
			(:p1, :p2, :p3, :p4, :p5)
		RETURNING id, datpri, datizm INTO :out0, :out1, :out2
	`

	now := time.Now()
	if user.DateOfCreation.IsZero() {
		user.DateOfCreation = now
	}
	if user.DateOfLastUpdate.IsZero() {
		user.DateOfLastUpdate = now
	}

	result, err := r.DB.ExecContext(ctx, query,
		sql.Named("p1", user.ADUsername),
		sql.Named("p2", user.FullName),
		sql.Named("p3", user.Status),
		sql.Named("p4", user.DateOfCreation),
		sql.Named("p5", user.DateOfLastUpdate),
		sql.Named("out0", sql.Out{Dest: &user.ID}),
		sql.Named("out1", sql.Out{Dest: &user.DateOfCreation}),
		sql.Named("out2", sql.Out{Dest: &user.DateOfLastUpdate}),
	)
	if err == nil {
		affected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			err = rowsErr
		} else if affected == 0 {
			err = sql.ErrNoRows
		}
	}

	if err != nil {
		return nil, err
	}

	user.Username = user.ADUsername
	return user, nil
}
