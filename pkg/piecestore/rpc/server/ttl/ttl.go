// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package ttl

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3" // sqlite is weird and needs underscore

	"storj.io/storj/pkg/piecestore"
)

type TTL struct {
	db *sql.DB
}

func NewTTL(DBPath string) (*TTL, error) {
	db, err := sql.Open("sqlite3", DBPath)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS `ttl` (`hash` TEXT UNIQUE, `created` INT(10), `expires` INT(10));")
	if err != nil {
		return nil, err
	}

	return &TTL{db}, nil
}

// CheckEntries -- checks for and deletes expired TTL entries
func CheckEntries(dir string, rows *sql.Rows) error {

	for rows.Next() {
		var expHash string
		var expires int64

		err := rows.Scan(&expHash, &expires)
		if err != nil {
			return err
		}

		// delete file on local machine
		err = pstore.Delete(expHash, dir)
		if err != nil {
			return err
		}

		log.Printf("Deleted file: %s\n", expHash)
		if rows.Err() != nil {

			return rows.Err()
		}
	}

	return nil
}

// DBCleanup -- go routine to check ttl database for expired entries
// pass in database and location of file for deletion
func (ttl *TTL) DBCleanup(dir string) error {

	tickChan := time.NewTicker(time.Second * 5).C
	for {
		select {
		case <-tickChan:
			rows, err := ttl.db.Query(fmt.Sprintf("SELECT hash, expires FROM ttl WHERE expires < %d", time.Now().Unix()))
			if err != nil {
				return err
			}

			err = CheckEntries(dir, rows)
			if err != nil {
				rows.Close()
				return err
			}
			rows.Close()

			_, err = ttl.db.Exec(fmt.Sprintf("DELETE FROM ttl WHERE expires < %d", time.Now().Unix()))
			if err != nil {
				return err
			}
		}
	}
}

// AddTTLToDB -- Insert TTL into database by hash
func (ttl *TTL) AddTTLToDB(hash string, expiration int64) error {

	_, err := ttl.db.Exec(fmt.Sprintf(`INSERT INTO ttl (hash, created, expires) VALUES ("%s", "%d", "%d")`, hash, time.Now().Unix(), expiration))
	if err != nil {
		return err
	}

	return nil
}

// GetTTLByHash -- Find the TTL in the database by hash and return it
func (ttl *TTL) GetTTLByHash(hash string) (expiration int64, err error) {

	rows, err := ttl.db.Query(fmt.Sprintf(`SELECT expires FROM ttl WHERE hash="%s"`, hash))
	if err != nil {
		return 0, err
	}

	for rows.Next() {
		err = rows.Scan(&expiration)
		if err != nil {
			return 0, err
		}
	}

	return expiration, nil
}

// DeleteTTLByHash -- Find the TTL in the database by hash and delete it
func (ttl *TTL) DeleteTTLByHash(hash string) error {

	_, err := ttl.db.Exec(fmt.Sprintf(`DELETE FROM ttl WHERE hash="%s"`, hash))
	return err
}