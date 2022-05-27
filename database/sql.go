package database

import (
	"database/sql"
	"time"
)

type jailedUser struct {
	id          uint64
	release     bool
	releaseTime time.Time
	reason      string
	oldnick     string
	oldroles    string
	jailrole    string
}

const create string = `
CREATE TABLE IF NOT EXISTS jailed (
id INTEGER NOT NULL PRIMARY KEY,
release INTEGER NOT NULL,
releasetime DATETIME NOT NULL,
reason TEXT,
oldnick TEXT NOT NULL,
oldroles TEXT,
jailrole TEXT
);`

func init() {
	db, err := sql.Open("sqlite3", "jail.db")
}
