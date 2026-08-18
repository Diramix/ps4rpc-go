package ps4info

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func openDB(data []byte) (*sql.DB, func(), error) {
	f, err := os.CreateTemp("", "ps4info-*.db")
	if err != nil {
		return nil, func() {}, err
	}
	name := f.Name()
	cleanup := func() { os.Remove(name) }

	if _, err := f.Write(data); err != nil {
		f.Close()
		cleanup()
		return nil, func() {}, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return nil, func() {}, err
	}

	db, err := sql.Open("sqlite", "file:"+name+"?mode=ro")
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	full := func() {
		db.Close()
		cleanup()
	}
	return db, full, nil
}
